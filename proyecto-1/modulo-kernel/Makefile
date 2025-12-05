# Definimos los dos objetos que queremos construir
obj-m += ram.o
obj-m += procesos.o

all:
	make -C /lib/modules/$(shell uname -r)/build M=$(PWD) modules

clean:
	make -C /lib/modules/$(shell uname -r)/build M=$(PWD) clean