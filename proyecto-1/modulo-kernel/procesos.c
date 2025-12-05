#include <linux/module.h>
#include <linux/kernel.h>
#include <linux/init.h>
#include <linux/proc_fs.h>
#include <linux/seq_file.h>
#include <linux/sched.h> // Importante: Contiene task_struct (la info del proceso)
#include <linux/mm.h>    // Importante: Para medir la memoria
#include <linux/sched/signal.h> // Para for_each_process

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Enner Mendizabal");
MODULE_DESCRIPTION("Monitor de Procesos SO1");
MODULE_VERSION("1.0");

#define PROCFS_NAME "sysinfo_so1_202302220"

static int my_proc_show(struct seq_file *m, void *v) {
    struct task_struct *task;
    unsigned long rss;
    unsigned long vsz;
    bool first = true;

    // Iniciamos el JSON
    seq_printf(m, "[\n");

    for_each_process(task) {
        // Solo procesos con memoria asignada
        if (task->mm) {
            // 1. CÁLCULO DE MEMORIA EN KB
            // PAGE_SHIFT suele ser 12 (4096 bytes). 
            // Para pasar a KB (1024 bytes), restamos 10 bits al shift.
            // Es decir: (Páginas * 4096) / 1024
            rss = get_mm_rss(task->mm) << (PAGE_SHIFT - 10);
            vsz = (task->mm->total_vm) << (PAGE_SHIFT - 10);

            // Manejo de comas
            if (!first) {
                seq_printf(m, ",\n");
            }
            first = false;

            // 2. IMPRESIÓN CON NUEVOS CAMPOS
            seq_printf(m, "  {\n");
            seq_printf(m, "    \"pid\": %d,\n", task->pid);
            seq_printf(m, "    \"name\": \"%s\",\n", task->comm);
            seq_printf(m, "    \"state\": %u,\n", task->__state);
            
            // Nuevos datos requeridos por el PDF:
            seq_printf(m, "    \"ram_kb\": %lu,\n", rss);      // RSS en KB
            seq_printf(m, "    \"vsz_kb\": %lu,\n", vsz);      // VSZ en KB
            
            // Datos crudos para que Go calcule el % de CPU
            // utime = tiempo en modo usuario, stime = tiempo en modo kernel
            seq_printf(m, "    \"cpu_utime\": %llu,\n", task->utime);
            seq_printf(m, "    \"cpu_stime\": %llu\n", task->stime);
            
            seq_printf(m, "  }");
        }
    }

    seq_printf(m, "\n]\n");
    return 0;
}

// Proceso de apertura del archivo /proc
static int my_proc_open(struct inode *inode, struct file *file) {
    return single_open(file, my_proc_show, NULL);
}

// Definimos las operaciones del archivo /proc
static const struct proc_ops my_proc_ops = {
    .proc_open = my_proc_open,
    .proc_read = seq_read,
    .proc_lseek = seq_lseek,
    .proc_release = single_release,
};

// Función de inicialización del módulo
static int __init my_module_init(void) {
    struct proc_dir_entry *entry;
    entry = proc_create(PROCFS_NAME, 0444, NULL, &my_proc_ops);
    if (!entry) {
        return -ENOMEM;
    }
    printk(KERN_INFO "SO1: Modulo de procesos cargado.\n");
    return 0;
}

// Función de limpieza del módulo
static void __exit my_module_exit(void) {
    remove_proc_entry(PROCFS_NAME, NULL);
    printk(KERN_INFO "SO1: Modulo de procesos descargado.\n");
}


module_init(my_module_init);
module_exit(my_module_exit);