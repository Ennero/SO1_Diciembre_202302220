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
// Nombres exactos definidos en tus archivos .c
const RAM_FILE = "/proc/continfo_so1_202302220"
const PROC_FILE = "/proc/sysinfo_so1_202302220" 
const DB_FILE = "./metrics.db"

const DESIRED_LOW = 3
const DESIRED_HIGH = 2

// --- ESTRUCTURAS ---

// Estructura para leer el JSON del Módulo RAM
type SystemRam struct {
	TotalMB    int `json:"total_ram_mb"`
	FreeMB     int `json:"free_ram_mb"`
	UsedMB     int `json:"used_ram_mb"`
	Percentage int `json:"percentage"`
}

// Estructura para leer el JSON del Módulo Procesos
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
	fmt.Println("--- Iniciando Daemon SO1 (Doble Módulo) ---")
	
	initDB()
	defer db.Close()

	fmt.Println("Monitor RAM:", RAM_FILE)
	fmt.Println("Monitor Procesos:", PROC_FILE)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("\n------------------------------------------------")
		fmt.Printf("[%s] Escaneando sistema...\n", time.Now().Format("15:04:05"))
		loop()
	}
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", DB_FILE)
	if err != nil {
		fmt.Println("Error fatal abriendo la BD:", err)
		os.Exit(1)
	}

	os.Chmod(DB_FILE, 0666)

	// Tabla 1: Histórico de RAM Global
	q1 := `CREATE TABLE IF NOT EXISTS ram_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		total INTEGER,
		used INTEGER,
		percentage INTEGER
	);`
	db.Exec(q1)

	// Tabla 2: Histórico de Procesos (Contenedores)
	q2 := `CREATE TABLE IF NOT EXISTS process_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		pid INTEGER,
		name TEXT,
		ram INTEGER,
		cpu REAL
	);`
	db.Exec(q2)
	
	fmt.Println("Base de datos lista: metrics.db")
}

func loop() {
	// --- PARTE A: LEER Y GUARDAR RAM GLOBAL ---
	ramData, err := readRamModule()
	if err != nil {
		fmt.Printf("⚠️ Error leyendo RAM (%s): %v\n", RAM_FILE, err)
	} else {
		fmt.Printf("💾 RAM SYSTEM: %d%% Usado (%d/%d MB)\n", ramData.Percentage, ramData.UsedMB, ramData.TotalMB)
		insertRamLog(ramData)
	}

	// --- PARTE B: PROCESOS, DOCKER Y THANOS ---
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
	now := time.Now() // Fecha unificada para los registros

	for _, proc := range kernelProcs {
		isHigh := false
		isLow := false

		// 1. Identificar si es un contenedor nuestro
		if strings.Contains(proc.Name, "stress") {
			if proc.RamKB > 50000 { isHigh = true } else { isHigh = true }
		} else if strings.Contains(proc.Name, "sleep") {
			isLow = true
		}

		// 2. Si es contenedor, procesar
		if isHigh || isLow {
			cpuPercent := calculateCPU(proc)
			ramMB := int(proc.RamKB / 1024)

			tipo := "BAJO"
			if isHigh { tipo = "ALTO" }
			
			fmt.Printf(" -> [%s] PID %d | RAM: %d MB | CPU: %.2f%%\n", 
				tipo, proc.Pid, ramMB, cpuPercent)
			
			// Guardar en BD
			insertProcessLog(now, proc.Pid, proc.Name, ramMB, cpuPercent)

			if isHigh {
				countHigh++
				procsHigh = append(procsHigh, proc)
			} else {
				countLow++
				procsLow = append(procsLow, proc)
			}
		}
	}

	fmt.Printf("RESUMEN CONTENEDORES: Altos: %d | Bajos: %d\n", countHigh, countLow)

	// 3. Lógica Thanos (Solo si Docker funciona y hay exceso)
	if len(dockerContainers) > 0 {
		if countHigh > DESIRED_HIGH {
			fmt.Printf("⚠️ Exceso de ALTOS (%d > %d). Eliminando...\n", countHigh, DESIRED_HIGH)
			killContainers(countHigh - DESIRED_HIGH, procsHigh)
		}
		if countLow > DESIRED_LOW {
			fmt.Printf("⚠️ Exceso de BAJOS (%d > %d). Eliminando...\n", countLow, DESIRED_LOW)
			killContainers(countLow - DESIRED_LOW, procsLow)
		}
	}
}

// --- LECTURAS DE ARCHIVOS /PROC ---

func readRamModule() (SystemRam, error) {
	var stats SystemRam
	data, err := os.ReadFile(RAM_FILE)
	if err != nil { return stats, err }
	err = json.Unmarshal(data, &stats)
	return stats, err
}

func readProcessModule() ([]KernelProcess, error) {
	data, err := os.ReadFile(PROC_FILE)
	if err != nil { return nil, err }
	var procs []KernelProcess
	err = json.Unmarshal(data, &procs)
	return procs, err
}

// --- BASES DE DATOS ---

func insertRamLog(ram SystemRam) {
	stmt, err := db.Prepare("INSERT INTO ram_log(total, used, percentage) VALUES(?, ?, ?)")
	if err != nil { fmt.Println("Error prep RAM:", err); return }
	defer stmt.Close()
	stmt.Exec(ram.TotalMB, ram.UsedMB, ram.Percentage)
}

func insertProcessLog(ts time.Time, pid int, name string, ram int, cpu float64) {
	stmt, err := db.Prepare("INSERT INTO process_log(timestamp, pid, name, ram, cpu) VALUES(?, ?, ?, ?, ?)")
	if err != nil { fmt.Println("Error prep PROC:", err); return }
	defer stmt.Close()
	stmt.Exec(ts, pid, name, ram, cpu)
}

// --- AUXILIARES (Docker, CPU, Kill) ---

func getDockerContainers() map[string]string {
	cmd := exec.Command("docker", "ps", "--format", "{{.ID}}|{{.Names}}|{{.Command}}")
	output, err := cmd.Output()
	if err != nil { return nil } // Docker no responde o no hay containers

	containers := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" { continue }
		parts := strings.Split(line, "|")
		if len(parts) >= 3 {
			id := parts[0]
			name := parts[1]
			containers[name] = id
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

	if deltaTime == 0 { return 0.0 }
	return (float64(deltaCpu) / 100.0) / deltaTime * 100
}

func killContainers(amount int, procs []KernelProcess) {
	killed := 0
	for _, proc := range procs {
		if killed >= amount { break }
		fmt.Printf("   💀 Matando PID %d (%s)...\n", proc.Pid, proc.Name)
		exec.Command("kill", "-9", fmt.Sprintf("%d", proc.Pid)).Run()
		killed++
	}
}