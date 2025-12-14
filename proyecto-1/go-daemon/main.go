package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// --- CONFIGURACIÓN ---
const RAM_FILE = "/proc/continfo_so1_202302220"
const PROC_FILE = "/proc/sysinfo_so1_202302220"
const DB_FILE = "./metrics.db"

// Rutas relativas desde la carpeta go-daemon
const GENERATOR_SCRIPT = "../bash/generator.sh"

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
	fmt.Println("--- Iniciando Daemon ---")

	// Inicializar BD
	initDB()
	defer db.Close()

	// Cargar Módulos de Kernel
	loadKernelModules()
	// -------------------------------------------

	fmt.Println("Monitor RAM:", RAM_FILE)
	fmt.Println("Monitor Procesos:", PROC_FILE)

	// Levantar Grafana
	startGrafanaService()

	setupCronjob()
	// ----------------------------------------------

	// Manejar señales para limpiar cron al salir
	setupSignalHandler()

	// Configurar Timers
	monitorTicker := time.NewTicker(20 * time.Second)
	defer monitorTicker.Stop()


	fmt.Println("Sistema corriendo. Presiona Ctrl+C para detener.")

	// Loop inicial inmediato
	for range monitorTicker.C {
		fmt.Println("\n------------------------------------------------")
		fmt.Printf("[%s] Escaneando sistema...\n", time.Now().Format("15:04:05"))
		loop()
	}
}

// Crear y preparar la base de datos SQLite
func initDB() {
	var err error
	db, err = sql.Open("sqlite3", DB_FILE)
	if err != nil {
		fmt.Println("Error fatal abriendo la BD:", err)
		os.Exit(1)
	}

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

	fmt.Println("Base de datos lista: metrics.db")
}

func startGrafanaService() {
	fmt.Println("Levantando Grafana via 'docker run'...")

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error obteniendo directorio actual:", err)
		return
	}

	// Ruta absoluta a metrics.db
	dbPath := fmt.Sprintf("%s/metrics.db", cwd)

	// Ruta a la carpeta de datos persistentes de Grafana
	dashboardPath := fmt.Sprintf("%s/../dashboard", cwd)
	grafanaDataPath := fmt.Sprintf("%s/grafana_data", dashboardPath)

	// Crear carpeta de datos si no existe
	if _, err := os.Stat(grafanaDataPath); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(grafanaDataPath, 0777); mkErr != nil {
			fmt.Printf("No se pudo crear grafana_data: %v\n", mkErr)
		}
	}

	// Asegurar permisos amplios (para que el contenedor pueda escribir)
	os.Chmod(grafanaDataPath, 0777)

	// Eliminar contenedor previo si existe
	exec.Command("docker", "rm", "-f", "grafana_so1").Run()

	// Levantar Grafana con volúmenes persistentes
	cmd := exec.Command(
		"docker", "run", "-d",
		"--name", "grafana_so1",
		"-p", "3000:3000",
		"-e", "GF_INSTALL_PLUGINS=frser-sqlite-datasource",
		// metrics.db en modo lectura
		"-v", fmt.Sprintf("%s:/var/lib/grafana/metrics.db:ro", dbPath),
		// carpeta de datos persistente
		"-v", fmt.Sprintf("%s:/var/lib/grafana", grafanaDataPath),
		"grafana/grafana:latest",
	)

	// Ejecutar comando
	output, err := cmd.CombinedOutput()

	// Reportar resultado
	if err != nil {
		fmt.Printf("Error levantando Grafana: %v\nSalida: %s\n", err, string(output))
	} else {
		containerID := strings.TrimSpace(string(output))
		if len(containerID) > 12 {
			containerID = containerID[:12]
		}
		fmt.Printf("Grafana iniciado (ID: %s). http://localhost:3000\n", containerID)
		fmt.Printf("Datos persistentes en: %s\n", grafanaDataPath)
	}
}

func loop() {
	ramData, err := readRamModule()
	if err != nil {
		fmt.Printf("Error leyendo RAM: %v\n", err)
	} else {
		fmt.Printf("RAM SYSTEM: %d%% Usado (%d/%d MB)\n", ramData.Percentage, ramData.UsedMB, ramData.TotalMB)
		insertRamLog(ramData)
	}

	// Obtener contenedores Docker activos
	dockerContainers := getDockerContainers()
	kernelProcs, err := readProcessModule()
	if err != nil {
		fmt.Printf("Error leyendo Procesos: %v\n", err)
		return
	}

	countLow := 0
	countHigh := 0

	// Listas para candidatos a matar
	var procsLow []KernelProcess
	var procsHigh []KernelProcess

	now := time.Now()

	for _, proc := range kernelProcs {
		isHighFamily := strings.Contains(proc.Name, "stress")
		isLowFamily := strings.Contains(proc.Name, "sleep")

		if isHighFamily || isLowFamily {
			cpuPercent := calculateCPU(proc)
			ramMB := int(proc.RamKB / 1024)

			insertProcessLog(now, proc.Pid, proc.Name, ramMB, cpuPercent)

			tipo := "BAJO"
			if isHighFamily {
				tipo = "ALTO"
				if proc.Name == "stress-ng" {
					countHigh++
				}
				procsHigh = append(procsHigh, proc)

			} else {
				// Para sleep, asumimos que no hay hijos complejos
				if proc.Name == "sleep" {
					countLow++
				}
				procsLow = append(procsLow, proc)
			}

			// Solo imprimimos en consola si tiene consumo relevante o es padre, para no ensuciar el log
			if cpuPercent > 0.1 || proc.Name == "stress-ng" || proc.Name == "sleep" {
				fmt.Printf(" -> [%s] PID %d (%s) | RAM: %d MB | CPU: %.2f%%\n", tipo, proc.Pid, proc.Name, ramMB, cpuPercent)
			}
		}
	}

	fmt.Printf("RESUMEN CONTENEDORES: Altos: %d/%d | Bajos: %d/%d\n", countHigh, DESIRED_HIGH, countLow, DESIRED_LOW)

	// Solo ejecutamos lógica de matanza si Docker está activo
	if len(dockerContainers) > 0 {
		if countHigh > DESIRED_HIGH {
			diff := countHigh - DESIRED_HIGH
			fmt.Printf("Exceso ALTOS (%d detectados). Eliminando %d...\n", countHigh, diff)
			// Priorizamos matar procesos padres ("stress-ng") primero
			killContainers(diff, procsHigh, "EXCESO_ALTO", "stress-ng")
		}
		if countLow > DESIRED_LOW {
			diff := countLow - DESIRED_LOW
			fmt.Printf("Exceso BAJOS (%d detectados). Eliminando %d...\n", countLow, diff)
			killContainers(diff, procsLow, "EXCESO_BAJO", "sleep")
		}
	}
}

// --- BASE DE DATOS ---

// Loguear en ram_log
func insertRamLog(ram SystemRam) {
	stmt, _ := db.Prepare("INSERT INTO ram_log(total, used, percentage) VALUES(?, ?, ?)")
	defer stmt.Close()
	stmt.Exec(ram.TotalMB, ram.UsedMB, ram.Percentage)
}

// Loguear en process_log
func insertProcessLog(ts time.Time, pid int, name string, ram int, cpu float64) {
	stmt, _ := db.Prepare("INSERT INTO process_log(timestamp, pid, name, ram, cpu) VALUES(?, ?, ?, ?, ?)")
	defer stmt.Close()
	stmt.Exec(ts, pid, name, ram, cpu)
}

// Loguear en kill_log
func insertKillLog(pid int, name string, reason string) {
	stmt, err := db.Prepare("INSERT INTO kill_log(pid, name, reason) VALUES(?, ?, ?)")
	if err != nil {
		fmt.Println("Error logueando kill:", err)
		return
	}
	defer stmt.Close()
	stmt.Exec(pid, name, reason)
}

// --- LÓGICA DE MATANZA ---

// Se añade filtro para intentar matar primero a los padres
func killContainers(amount int, procs []KernelProcess, reason string, targetName string) {
	killed := 0

	// Matar coincidencias exactas
	for _, proc := range procs {
		if killed >= amount {
			return
		}
		if proc.Name == targetName {
			fmt.Printf("   💀 Matando PID %d (%s)...\n", proc.Pid, proc.Name)
			insertKillLog(proc.Pid, proc.Name, reason)
			exec.Command("kill", "-9", fmt.Sprintf("%d", proc.Pid)).Run()
			killed++
		}
	}

	// Si aun falta por matar, matar cualquier cosa de la lista
	for _, proc := range procs {
		if killed >= amount {
			return
		}
		// Matar si no coincide con targetName (ya que esos ya murieron o no estaban)
		if proc.Name != targetName {
			fmt.Printf("   💀 Matando PID %d (%s) [Limpieza]...\n", proc.Pid, proc.Name)
			insertKillLog(proc.Pid, proc.Name, reason)
			exec.Command("kill", "-9", fmt.Sprintf("%d", proc.Pid)).Run()
			killed++
		}
	}
}

// Leer estadísticas de RAM desde el módulo del kernel
func readRamModule() (SystemRam, error) {
	var stats SystemRam
	data, err := os.ReadFile(RAM_FILE)
	if err != nil {
		return stats, err
	}
	err = json.Unmarshal(data, &stats)
	return stats, err
}

// Leer lista de procesos desde el módulo del kernel
func readProcessModule() ([]KernelProcess, error) {
	data, err := os.ReadFile(PROC_FILE)
	if err != nil {
		return nil, err
	}
	var procs []KernelProcess
	err = json.Unmarshal(data, &procs)
	return procs, err
}

// Obtener lista de contenedores Docker activos
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

// Calcular uso de CPU basado en historial
func calculateCPU(proc KernelProcess) float64 {
	currentTotalTime := proc.CpuUtime + proc.CpuStime
	currentTime := time.Now()

	// Obtener stats previos
	stats, exists := history[proc.Pid]
	if !exists {
		history[proc.Pid] = ProcessStats{
			Pid:       proc.Pid,
			TotalTime: currentTotalTime,
			LastSeen:  currentTime,
		}
		return 0.0
	}

	// Calcular diferencias
	deltaCpu := currentTotalTime - stats.TotalTime
	deltaTime := currentTime.Sub(stats.LastSeen)

	// Actualizar historial
	history[proc.Pid] = ProcessStats{
		Pid:       proc.Pid,
		TotalTime: currentTotalTime,
		LastSeen:  currentTime,
	}

	// Evitar divisiones raras
	if deltaTime <= 0 || deltaCpu == 0 {
		return 0.0
	}

	// CPU% = (cpu_time_interval / real_interval) * 100 (lo da en nanosegundos)
	cpuUsage := (float64(deltaCpu) / float64(deltaTime.Nanoseconds())) * 100.0

	// Filtrar valores absurdos por errores de medición o reinicios
	if cpuUsage < 0 || math.IsNaN(cpuUsage) || math.IsInf(cpuUsage, 0) {
		return 0.0
	}

	if cpuUsage > 400.0 {
		cpuUsage = 400.0
	}

	return cpuUsage
}

// Configurar manejador de señales para limpiar cronjob al salir
func setupSignalHandler() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-stop
		fmt.Println("\nDeteniendo servicio...")

		// Limpiar Cronjob
		fmt.Println("Limpiando cronjob...")
		cmd := exec.Command("bash", "-c",
			`crontab -l 2>/dev/null | grep -v 'bash/generator.sh' | crontab -`)
		cmd.Run()

		// Descargar Módulos
		fmt.Println("Descargando módulos...")
		exec.Command("sudo", "rmmod", "sysinfo").Run()
		exec.Command("sudo", "rmmod", "continfo").Run()

		os.Exit(0)
	}()
}

// Cargar módulos del kernel
func loadKernelModules() {
	fmt.Println("Cargando módulos del kernel...")
	cmd1 := exec.Command("sudo", "insmod", "../modulo-kernel/sysinfo.ko")
	cmd1.Run()
	cmd2 := exec.Command("sudo", "insmod", "../modulo-kernel/continfo.ko")
	cmd2.Run()

	fmt.Println("Módulos cargados")
}

// Configurar cronjob para ejecutar generator.sh cada minuto
func setupCronjob() {
	fmt.Println("Configurando Cronjob del sistema...")

	cwd, _ := os.Getwd()
	scriptPath := fmt.Sprintf("%s/../bash/generator.sh", cwd)

	cronEntry := fmt.Sprintf("* * * * * /bin/bash %s", scriptPath)

	cmd := exec.Command("bash", "-c",
		fmt.Sprintf("(crontab -l 2>/dev/null; echo \"%s\") | crontab -", cronEntry))

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error creando cronjob: %v | %s\n", err, string(output))
	} else {
		fmt.Println("Cronjob configurado en el sistema operativo.")
	}
}
