# Changelog generator

Tool that can be used to generate a CHANGELOG file from individual change
files written in YAML.

Usage:

```sh
    # generates a new change YAML file from a template
    chloggen new -filename <filename>
    # validates all change YAML files
    chloggen validate
    # provide a preview of the generated changelog file
    chloggen update -dry
    # updates the changelog file
    chloggen update -version <version>
```
