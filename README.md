# Terraform Cloud Audit Trail Viewer

A CLI tool to view Terraform Cloud audit trails.

Uses <https://github.com/rivo/tview> for the interface.

## Building the Tool

```bash
go build -o tfaudit
```

## Usage

```bash
./tfaudit --t $TF_ORG_TOKEN --since 1
```
