#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/init.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/mm.h>
#include <linux/sysinfo.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Enner Mendizabal");
MODULE_DESCRIPTION("Modulo RAM/Contenedores SO1 - ContInfo");
MODULE_VERSION("1.0");

// REQUISITO: El módulo de memoria/contenedores va en continfo
// Asegúrate de cambiar el carnet
#define PROCFS_NAME "continfo_so1_202302220"

static int my_proc_show(struct seq_file *m, void *v) {
    struct sysinfo si;
    
    si_meminfo(&si);
    
    // Convertir a MB
    unsigned long total_ram = (si.totalram * si.mem_unit) / (1024 * 1024);
    unsigned long free_ram = (si.freeram * si.mem_unit) / (1024 * 1024);
    unsigned long buffer_ram = (si.bufferram * si.mem_unit) / (1024 * 1024);
    // Linux calcula 'used' restando free y buffers/cache del total
    unsigned long used_ram = total_ram - free_ram - buffer_ram;
    
    unsigned long percent = 0;
    if (total_ram > 0) {
        percent = (used_ram * 100) / total_ram;
    }

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
    printk(KERN_INFO "SO1: Modulo RAM (continfo) cargado.\n");
    return 0;
}

static void __exit my_module_exit(void) {
    remove_proc_entry(PROCFS_NAME, NULL);
    printk(KERN_INFO "SO1: Modulo RAM (continfo) descargado.\n");
}

module_init(my_module_init);
module_exit(my_module_exit);