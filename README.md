# efibootcfg 🥾

Manipulate the UEFI boot manager.

A userspace application used to modify the UEFI Boot Manager configuration.  This application can create and
destroy boot entries, change the boot order, change the next running boot option, and more.

It's meant to be a port of the tool [efibootmgr](https://github.com/rhboot/efibootmgr) in [Go](https://go.dev/) to demonstrate the capabilities of the Go package [github.com/0x5a17ed/uefi](https://github.com/0x5a17ed/uefi) and how to use it.


## 🎯 Goals 

The objective of this tool is to support configuring all aspects of the UEFI Boot Manager.


## 📦 Installation

```console
$ go get -u github.com/0x5a17ed/efibootcfg@latest
```


## 🤔 Usage

```console
foo@bar:~ $ efibootcfg
BootCurrent: 0001
BootOrder:   {0001, 0000}
Boot0001*:   "ArchLinux"
Boot0000*:   "Windows Boot Manager"
```


## 💡 Features

- can read uefi boot manager load options.


## ☝️ Is it any good?

[yes](https://news.ycombinator.com/item?id=3067434).
