To generate the manifests from this directory, run the following

```
jb update

tk export out_dir generator --format "{{.kind}}-{{or .metadata.name .metadata.generateName}}"
```

The manifests will be generated in the out_dir.