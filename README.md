ovsdpdk-network-operator (In Progress)
=======================

# Build Container Image
Two images are built `ovsdpdk-network-operator` and `operator-network-prepare`
in this operator, which below command.

```
make dev
```

# Generate
In case of type changes to the operator API, use below command to generate the
crds and deepcopy files.

```
make geneate
```

# Build
```
make
```

# Test
```
make test
```

