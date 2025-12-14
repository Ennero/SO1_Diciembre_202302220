#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/init.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/sched.h>
#include <linux/mm.h>
#include <linux/sched/signal.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Enner Mendizabal");
MODULE_DESCRIPTION("Monitor de Procesos SO1 - SysInfo");
MODULE_VERSION("1.0");

// Asegúrate de cambiar el carnet
#define PROCFS_NAME "sysinfo_so1_202302220"

// Función para mostrar la información en el archivo proc
static int my_proc_show(struct seq_file *m, void *v) {
    struct task_struct *task;
    unsigned long rss;
    unsigned long vsz;
    unsigned long total_ram_pages;
    bool first = true;

    // Obtenemos la RAM total del sistema en páginas para calcular el %
    total_ram_pages = totalram_pages();
    if (total_ram_pages == 0) total_ram_pages = 1; // Evitar división por cero

    seq_printf(m, "[\n");

    for_each_process(task) {
        if (task->mm) {
            // rss devuelve páginas, convertimos a KB (paginas * 4)
            rss = get_mm_rss(task->mm) << (PAGE_SHIFT - 10);
            vsz = (task->mm->total_vm) << (PAGE_SHIFT - 10);

            // Cálculo del porcentaje de RAM (RSS / Total RAM)
            unsigned long rss_pages = get_mm_rss(task->mm);
            unsigned long mem_percent = (rss_pages * 100) / total_ram_pages;

            if (!first) {
                seq_printf(m, ",\n");
            }
            first = false;

            seq_printf(m, "  {\n");
            seq_printf(m, "    \"pid\": %d,\n", task->pid);
            seq_printf(m, "    \"name\": \"%s\",\n", task->comm);
            // El estado suele ser long, usamos %ld o %u según kernel
            seq_printf(m, "    \"state\": %u,\n", task->__state);
            
            seq_printf(m, "    \"ram_kb\": %lu,\n", rss);
            seq_printf(m, "    \"vsz_kb\": %lu,\n", vsz);
            // Cumpliendo requisito de porcentaje de memoria
            seq_printf(m, "    \"ram_percent\": %lu,\n", mem_percent);
            
            // Dejamos CPU crudo para calcular en Frontend
            seq_printf(m, "    \"cpu_utime\": %llu,\n", task->utime);
            seq_printf(m, "    \"cpu_stime\": %llu\n", task->stime);
            
            seq_printf(m, "  }");
        }
    }

    seq_printf(m, "\n]\n");
    return 0;
}

// Apertura del archivo proc
static int my_proc_open(struct inode *inode, struct file *file) {
    return single_open(file, my_proc_show, NULL);
}

// Definición de las operaciones del archivo proc
static const struct proc_ops my_proc_ops = {
    .proc_open = my_proc_open,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = single_release,
};

// Inicialización del módulo
static int __init my_module_init(void) {
    proc_create(PROCFS_NAME, 0444, NULL, &my_proc_ops);
    printk(KERN_INFO "SO1: Modulo Procesos (sysinfo) cargado.\n");
    return 0;
}

// Limpieza del módulo
static void __exit my_module_exit(void) {
    remove_proc_entry(PROCFS_NAME, NULL);
    printk(KERN_INFO "SO1: Modulo Procesos (sysinfo) descargado.\n");
}

module_init(my_module_init);
module_exit(my_module_exit);