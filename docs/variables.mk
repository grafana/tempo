# List of projects to provide to the make-docs script.
PROJECTS = tempo helm-charts/tempo-distributed loki

# The default version for grafana.com docs. Used by doc-validator for canonical checks.
# Can be empty for unversioned projects.
PRIMARY_PROJECT_VERSION := latest
