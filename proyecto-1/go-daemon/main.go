package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// --- CONFIGURACIÓN ---
// Cambia esto por el nombre real de tu archivo en /proc
const PROC_FILE = "/proc/continfo_so1_202302220" 
const DESIRED_LOW = 3  // Cantidad deseada de contenedores "Bajo Consumo"
const DESIRED_HIGH = 2 // Cantidad deseada de contenedores "Alto Consumo"

// --- ESTRUCTURAS ---

// 1. Estructura para leer el JSON del Kernel
type KernelProcess struct {
	Pid      int    `json:"pid"`
	Name     string `json:"name"`
	State    uint   `json:"state"`
	RamKB    uint64 `json:"ram_kb"`
	VszKB    uint64 `json:"vsz_kb"`
	CpuUtime uint64 `json:"cpu_utime"`
	CpuStime uint64 `json:"cpu_stime"`
}

// 2. Estructura interna para guardar el estado anterior (para calcular CPU)
type ProcessStats struct {
	Pid        int
	TotalTime  uint64 // utime + stime
	LastSeen   time.Time
}

// Mapa para recordar el historial de cada proceso
var history = make(map[int]ProcessStats)

func main() {
	fmt.Println("--- Iniciando Daemon SO1 ---")
	fmt.Println("Monitoreando archivo:", PROC_FILE)

	// Ciclo infinito cada 5 segundos
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("\n------------------------------------------------")
		fmt.Printf("[%s] Escaneando sistema...\n", time.Now().Format("15:04:05"))
		loop()
	}
}

func loop() {
	// 1. Obtener contenedores de Docker (ID y Nombre)
	// Usamos map para búsqueda rápida: map[NombreContenedor]ContainerID
	dockerContainers := getDockerContainers()
	if len(dockerContainers) == 0 {
		fmt.Println("No se detectaron contenedores Docker corriendo.")
		return
	}

	// 2. Leer datos del Kernel
	kernelProcs, err := readKernelProcs()
	if err != nil {
		fmt.Printf("Error leyendo Kernel: %v\n", err)
		return
	}

	// 3. Cruzar información: ¿Qué procesos del Kernel son Contenedores?
	// Contadores
	countLow := 0
	countHigh := 0
	
	// Listas para decidir a quién matar si sobran
	var procsLow []KernelProcess
	var procsHigh []KernelProcess

	for _, proc := range kernelProcs {
		// Buscamos si el nombre del proceso coincide con algún contenedor
		// NOTA: stress-ng suele lanzar procesos hijos, así que filtramos por nombre
		
		isHigh := false
		isLow := false

		// Estrategia simple de detección basada en tus Dockerfiles:
		// Si el proceso consume mucha RAM (> 50MB) es High
		// Si consume poca (< 10MB) es Low
		// (Ajusta estos umbrales según tu VM)
		if strings.Contains(proc.Name, "stress") {
			if proc.RamKB > 50000 { // 50 MB
				isHigh = true
			} else {
				// A veces stress-ng inicia con poca RAM, pero asumiremos High si es stress
				isHigh = true 
			}
		} else if strings.Contains(proc.Name, "sleep") {
			isLow = true
		}

		// Si encontramos uno de nuestros contenedores, calculamos CPU y clasificamos
		if isHigh || isLow {
			cpuPercent := calculateCPU(proc)
			
			// Imprimimos info bonita
			tipo := "BAJO"
			if isHigh { tipo = "ALTO" }
			fmt.Printf(" -> DETECTADO [%s]: PID %d | RAM: %d MB | CPU: %.2f%%\n", 
				tipo, proc.Pid, proc.RamKB/1024, cpuPercent)

			if isHigh {
				countHigh++
				procsHigh = append(procsHigh, proc)
			} else {
				countLow++
				procsLow = append(procsLow, proc)
			}
		}
	}

	fmt.Printf("\nRESUMEN: Altos: %d (Meta: %d) | Bajos: %d (Meta: %d)\n", 
		countHigh, DESIRED_HIGH, countLow, DESIRED_LOW)

	// 4. Lógica de Matanza (Thanos)
	// Si hay más de la cuenta, matamos al más reciente o al que más consuma.
	
	if countHigh > DESIRED_HIGH {
		diff := countHigh - DESIRED_HIGH
		fmt.Printf("⚠️ Sobran %d contenedores de ALTO consumo. Eliminando...\n", diff)
		killContainers(diff, procsHigh, dockerContainers)
	}

	if countLow > DESIRED_LOW {
		diff := countLow - DESIRED_LOW
		fmt.Printf("⚠️ Sobran %d contenedores de BAJO consumo. Eliminando...\n", diff)
		killContainers(diff, procsLow, dockerContainers)
	}
}

// --- FUNCIONES AUXILIARES ---

func readKernelProcs() ([]KernelProcess, error) {
	data, err := os.ReadFile(PROC_FILE)
	if err != nil {
		return nil, err
	}
	var procs []KernelProcess
	if err := json.Unmarshal(data, &procs); err != nil {
		return nil, err
	}
	return procs, nil
}

// Ejecuta "docker ps" para obtener IDs y Nombres reales
func getDockerContainers() map[string]string {
	cmd := exec.Command("docker", "ps", "--format", "{{.ID}}|{{.Names}}|{{.Command}}")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	containers := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" { continue }
		parts := strings.Split(line, "|")
		if len(parts) >= 3 {
			id := parts[0]
			name := parts[1]
			command := parts[2]
			
			// Guardamos una referencia.
			// La clave será útil para identificar qué es qué.
			if strings.Contains(command, "stress") {
				containers["stress"] = id // Simplificación
			} else if strings.Contains(command, "sleep") {
				containers["sleep"] = id
			}
			// Guardamos también por nombre exacto por si acaso
			containers[name] = id
		}
	}
	return containers
}

// Calcula el % de CPU comparando con la vez anterior
func calculateCPU(proc KernelProcess) float64 {
	currentTotalTime := proc.CpuUtime + proc.CpuStime
	currentTime := time.Now()

	stats, exists := history[proc.Pid]
	
	// Si es la primera vez que lo vemos, no podemos calcular % (necesitamos 2 puntos)
	if !exists {
		history[proc.Pid] = ProcessStats{
			Pid:       proc.Pid,
			TotalTime: currentTotalTime,
			LastSeen:  currentTime,
		}
		return 0.0
	}

	// Cálculo del Delta
	deltaCpu := currentTotalTime - stats.TotalTime
	deltaTime := currentTime.Sub(stats.LastSeen).Seconds() // En segundos

	// Actualizamos historial
	history[proc.Pid] = ProcessStats{
		Pid:       proc.Pid,
		TotalTime: currentTotalTime,
		LastSeen:  currentTime,
	}

	hertz := 100.0 
	cpuUsage := (float64(deltaCpu) / hertz) / deltaTime * 100

	return cpuUsage
}

// Mata los contenedores sobrantes
func killContainers(amount int, procs []KernelProcess, dockerMap map[string]string) {
	killed := 0
	for _, proc := range procs {
		if killed >= amount { break }
		
		// En Docker, matar el proceso principal dentro del contenedor suele detener el contenedor.
		fmt.Printf("   Matando proceso PID %d (%s)...\n", proc.Pid, proc.Name)
		
		// Opción A: Matar el proceso directamente (Más fácil y efectivo para la práctica)
		cmd := exec.Command("kill", "-9", fmt.Sprintf("%d", proc.Pid))
		cmd.Run()
		
		killed++
	}
}