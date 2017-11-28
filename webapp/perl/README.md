### dependencies

* GMP
```console
$ apt-get install libgmp-dev
```

### setup

```console
$ carton install
```

### dev
```console
$ carton exec morbo -l http://*:5000 main.pl
```

### prod
```console
$ carton exec hypnotoad -f main.pl
```
