recursive = true
output_file = "all.txt"
extensions = [".go"]
exclude_dirs  {
items = ["_examples", "_lab", "_tmp", "pkg", "lab","bin"]
}
exclude_files {
  items = ["before.txt","after.txt"]
}
use_gitignore = true
detailed = true
go_mode = "all"