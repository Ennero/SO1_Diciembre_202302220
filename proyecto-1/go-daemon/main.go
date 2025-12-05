package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// --- CONFIGURACIÓN ---
const RAM_FILE = "/proc/continfo_so1_202302220"
const PROC_FILE = "/proc/sysinfo_so1_202302220"
const DB_FILE = "./metrics.db"

// Rutas relativas desde la carpeta go-daemon
const GENERATOR_SCRIPT = "../bash/generator.sh"
const GRAFANA_COMPOSE = "../dashboard/docker-compose.yml"

const DESIRED_LOW = 3
const DESIRED_HIGH = 2

// --- ESTRUCTURAS ---

type SystemRam struct {
	TotalMB    int `json:"total_ram_mb"`
	FreeMB     int `json:"free_ram_mb"`
	UsedMB     int `json:"used_ram_mb"`
	Percentage int `json:"percentage"`
}

type KernelProcess struct {
	Pid      int    `json:"pid"`
	Name     string `json:"name"`
	State    uint   `json:"state"`
	RamKB    uint64 `json:"ram_kb"`
	VszKB    uint64 `json:"vsz_kb"`
	CpuUtime uint64 `json:"cpu_utime"`
	CpuStime uint64 `json:"cpu_stime"`
}

type ProcessStats struct {
	Pid       int
	TotalTime uint64
	LastSeen  time.Time
}

var history = make(map[int]ProcessStats)
var db *sql.DB

func main() {
	fmt.Println("--- Iniciando Daemon SO1 (Full Automático) ---")

	// 1. Inicializar BD (Vital hacerlo antes de Docker)
	initDB()
	defer db.Close()

	fmt.Println("Monitor RAM:", RAM_FILE)
	fmt.Println("Monitor Procesos:", PROC_FILE)

	// 2. Levantar Grafana Automáticamente
	startGrafanaService()

	// 3. Configurar Timers
	monitorTicker := time.NewTicker(5 * time.Second)
	defer monitorTicker.Stop()

	// Generador de tráfico (Cada 60s)
	generatorTicker := time.NewTicker(60 * time.Second)
	defer generatorTicker.Stop()

	// Carga inicial de tráfico
	go triggerTraffic()

	fmt.Println("✅ Sistema corriendo. Presiona Ctrl+C para detener.")

	// Bucle principal
	for {
		select {
		case <-monitorTicker.C:
			// Cada 5 seg: Escanear y Matar
			fmt.Println("\n------------------------------------------------")
			fmt.Printf("[%s] 🔍 Escaneando sistema...\n", time.Now().Format("15:04:05"))
			loop()

		case <-generatorTicker.C:
			// Cada 60 seg: Crear nuevos contenedores
			fmt.Println("\n------------------------------------------------")
			fmt.Printf("[%s] 🚀 Generando tráfico automático...\n", time.Now().Format("15:04:05"))
			go triggerTraffic()
		}
	}
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", DB_FILE)
	if err != nil {
		fmt.Println("Error fatal abriendo la BD:", err)
		os.Exit(1)
	}

	// Permisos vitales para que Grafana pueda leer el archivo
	os.Chmod(DB_FILE, 0666)

	// Tablas
	db.Exec(`CREATE TABLE IF NOT EXISTS ram_log (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
        total INTEGER,
        used INTEGER,
        percentage INTEGER
    );`)

	db.Exec(`CREATE TABLE IF NOT EXISTS process_log (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
        pid INTEGER,
        name TEXT,
        ram INTEGER,
        cpu REAL
    );`)

	db.Exec(`CREATE TABLE IF NOT EXISTS kill_log (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
        pid INTEGER,
        name TEXT,
        reason TEXT
    );`)

	fmt.Println("✅ Base de datos lista: metrics.db")
}

// --- FUNCIÓN NUEVA: LEVANTAR GRAFANA ---
func startGrafanaService() {
	fmt.Println("🐳 Intentando levantar Grafana con Docker Compose...")

	// Usamos la ruta relativa definida en la constante
	cmd := exec.Command("docker-compose", "-f", GRAFANA_COMPOSE, "up", "-d")
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Si falla docker-compose, intentamos con "docker compose" (versión nueva)
		fmt.Println("⚠️ 'docker-compose' falló, intentando 'docker compose'...")
		cmd = exec.Command("docker", "compose", "-f", GRAFANA_COMPOSE, "up", "-d")
		output, err = cmd.CombinedOutput() // Aquí reasignamos output
		if err != nil {
			fmt.Printf("❌ Error crítico levantando Grafana: %v\n", err)
			fmt.Println("Salida:", string(output)) // Aquí sí se usaba
			fmt.Println("➡️ INTENTA LEVANTARLO MANUALMENTE EN LA CARPETA DASHBOARD")
			return
		}
	}
	
	// --- CORRECCIÓN AQUÍ ---
	// Antes no usábamos 'output' si todo salía bien. Ahora lo imprimimos.
	fmt.Println("✅ Grafana levantado correctamente (localhost:3000)")
	fmt.Println("Detalles Docker:", string(output)) 
}

func triggerTraffic() {
	cmd := exec.Command("/bin/bash", GENERATOR_SCRIPT)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("⚠️ Error ejecutando generator.sh: %v\n", err)
		fmt.Println("Salida:", string(output))
	} else {
		fmt.Println("✅ Tráfico generado exitosamente.")
	}
}

func loop() {
	ramData, err := readRamModule()
	if err != nil {
		fmt.Printf("⚠️ Error leyendo RAM (%s): %v\n", RAM_FILE, err)
	} else {
		fmt.Printf("💾 RAM SYSTEM: %d%% Usado (%d/%d MB)\n", ramData.Percentage, ramData.UsedMB, ramData.TotalMB)
		insertRamLog(ramData)
	}

	dockerContainers := getDockerContainers()
	kernelProcs, err := readProcessModule()
	if err != nil {
		fmt.Printf("⚠️ Error leyendo Procesos (%s): %v\n", PROC_FILE, err)
		return
	}

	countLow := 0
	countHigh := 0
	var procsLow []KernelProcess
	var procsHigh []KernelProcess
	now := time.Now()

	for _, proc := range kernelProcs {
		isHigh := false
		isLow := false

		if strings.Contains(proc.Name, "stress") {
			isHigh = true
		} else if strings.Contains(proc.Name, "sleep") {
			isLow = true
		}

		if isHigh || isLow {
			cpuPercent := calculateCPU(proc)
			ramMB := int(proc.RamKB / 1024)

			insertProcessLog(now, proc.Pid, proc.Name, ramMB, cpuPercent)

			tipo := "BAJO"
			if isHigh {
				tipo = "ALTO"
				countHigh++
				procsHigh = append(procsHigh, proc)
			} else {
				countLow++
				procsLow = append(procsLow, proc)
			}

			fmt.Printf(" -> [%s] PID %d | RAM: %d MB | CPU: %.2f%%\n", tipo, proc.Pid, ramMB, cpuPercent)
		}
	}

	fmt.Printf("RESUMEN: Altos: %d/%d | Bajos: %d/%d\n", countHigh, DESIRED_HIGH, countLow, DESIRED_LOW)

	if len(dockerContainers) > 0 {
		if countHigh > DESIRED_HIGH {
			fmt.Printf("⚠️ Exceso ALTOS. Eliminando %d...\n", countHigh-DESIRED_HIGH)
			killContainers(countHigh-DESIRED_HIGH, procsHigh, "EXCESO_ALTO")
		}
		if countLow > DESIRED_LOW {
			fmt.Printf("⚠️ Exceso BAJOS. Eliminando %d...\n", countLow-DESIRED_LOW)
			killContainers(countLow-DESIRED_LOW, procsLow, "EXCESO_BAJO")
		}
	}
}

// --- BASE DE DATOS ---

func insertRamLog(ram SystemRam) {
	stmt, _ := db.Prepare("INSERT INTO ram_log(total, used, percentage) VALUES(?, ?, ?)")
    defer stmt.Close()
	stmt.Exec(ram.TotalMB, ram.UsedMB, ram.Percentage)
}

func insertProcessLog(ts time.Time, pid int, name string, ram int, cpu float64) {
	stmt, _ := db.Prepare("INSERT INTO process_log(timestamp, pid, name, ram, cpu) VALUES(?, ?, ?, ?, ?)")
    defer stmt.Close()
	stmt.Exec(ts, pid, name, ram, cpu)
}

func insertKillLog(pid int, name string, reason string) {
	stmt, err := db.Prepare("INSERT INTO kill_log(pid, name, reason) VALUES(?, ?, ?)")
	if err != nil {
		fmt.Println("Error logueando kill:", err)
		return
	}
    defer stmt.Close()
	stmt.Exec(pid, name, reason)
}

// --- AUXILIARES ---

func killContainers(amount int, procs []KernelProcess, reason string) {
	killed := 0
	for _, proc := range procs {
		if killed >= amount {
			break
		}
		fmt.Printf("   💀 Matando PID %d (%s)...\n", proc.Pid, proc.Name)
		insertKillLog(proc.Pid, proc.Name, reason)
		exec.Command("kill", "-9", fmt.Sprintf("%d", proc.Pid)).Run()
		killed++
	}
}

func readRamModule() (SystemRam, error) {
	var stats SystemRam
	data, err := os.ReadFile(RAM_FILE)
	if err != nil {
		return stats, err
	}
	err = json.Unmarshal(data, &stats)
	return stats, err
}

func readProcessModule() ([]KernelProcess, error) {
	data, err := os.ReadFile(PROC_FILE)
	if err != nil {
		return nil, err
	}
	var procs []KernelProcess
	err = json.Unmarshal(data, &procs)
	return procs, err
}

func getDockerContainers() map[string]string {
	cmd := exec.Command("docker", "ps", "--format", "{{.ID}}|{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return map[string]string{}
	}
	containers := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line != "" {
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				containers[parts[1]] = parts[0]
			}
		}
	}
	return containers
}

func calculateCPU(proc KernelProcess) float64 {
	currentTotalTime := proc.CpuUtime + proc.CpuStime
	currentTime := time.Now()
	stats, exists := history[proc.Pid]

	if !exists {
		history[proc.Pid] = ProcessStats{Pid: proc.Pid, TotalTime: currentTotalTime, LastSeen: currentTime}
		return 0.0
	}

	deltaCpu := currentTotalTime - stats.TotalTime
	deltaTime := currentTime.Sub(stats.LastSeen).Seconds()
	history[proc.Pid] = ProcessStats{Pid: proc.Pid, TotalTime: currentTotalTime, LastSeen: currentTime}

	if deltaTime == 0 {
		return 0.0
	}
	return (float64(deltaCpu) / 100.0) / deltaTime * 100
}