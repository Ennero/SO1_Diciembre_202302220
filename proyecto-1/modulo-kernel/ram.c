#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/init.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/mm.h>
#include <linux/sysinfo.h> // Necesario para obtener información del sistema

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Enner Mendizabal");
MODULE_DESCRIPTION("Modulo RAM SO1");
MODULE_VERSION("1.0");

// CAMBIAR CARNET
#define PROCFS_NAME "raminfo_so1_202302220"

static int my_proc_show(struct seq_file *m, void *v) {
    struct sysinfo si;
    
    // Obtener información del sistema
    si_meminfo(&si);
    
    // Convertir a Megabytes (MB)
    // sysinfo devuelve páginas, mem_unit es el tamaño en bytes de la página
    unsigned long total_ram = (si.totalram * si.mem_unit) / (1024 * 1024);
    unsigned long free_ram = (si.freeram * si.mem_unit) / (1024 * 1024);
    unsigned long buffer_ram = (si.bufferram * si.mem_unit) / (1024 * 1024);
    unsigned long shared_ram = (si.sharedram * si.mem_unit) / (1024 * 1024);
    
    // La RAM usada se suele calcular: Total - Libre - Buffers/Cache
    unsigned long used_ram = total_ram - free_ram - buffer_ram;
    
    // Calcular porcentaje
    unsigned long percent = (used_ram * 100) / total_ram;

    // Generar JSON simple (Objeto único, no array)
    seq_printf(m, "{\n");
    seq_printf(m, "  \"total_ram_mb\": %lu,\n", total_ram);
    seq_printf(m, "  \"free_ram_mb\": %lu,\n", free_ram);
    seq_printf(m, "  \"used_ram_mb\": %lu,\n", used_ram);
    seq_printf(m, "  \"percentage\": %lu\n", percent);
    seq_printf(m, "}\n");

    return 0;
}

static int my_proc_open(struct inode *inode, struct file *file) {
    return single_open(file, my_proc_show, NULL);
}

static const struct proc_ops my_proc_ops = {
    .proc_open = my_proc_open,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = single_release,
};

static int __init my_module_init(void) {
    proc_create(PROCFS_NAME, 0444, NULL, &my_proc_ops);
    printk(KERN_INFO "SO1: Modulo RAM cargado.\n");
    return 0;
}

static void __exit my_module_exit(void) {
    remove_proc_entry(PROCFS_NAME, NULL);
    printk(KERN_INFO "SO1: Modulo RAM descargado.\n");
}

module_init(my_module_init);
module_exit(my_module_exit);