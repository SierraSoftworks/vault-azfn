resource "null_resource" "vault_binary" {
    triggers = {
        vault_version = var.vault_version
    }

    provisioner "local-exec" {
        command = <<EOH
set -e
curl -o vault.zip https://releases.hashicorp.com/vault/${var.vault_version}/vault_${var.vault_version}_linux_amd64.zip
unzip vault.zip
rm vault.zip
chmod 0755 vault
mv vault ${path.module}/../files/vault
EOH
    }
}

resource "null_resource" "agent_binary" {
    triggers = {
        every_run = timestamp()
    }

    provisioner "local-exec" {
        command = <<EOH
set -e
export GOOS=linux
export GOARCH=amd64
go build -o '${path.module}/../files/vault-launcher' '${path.module}/../main.go'
EOH
    }
}

data "archive_file" "server" {
  depends_on = [null_resource.vault_binary, null_resource.agent_binary]

  type        = "zip"
  output_path = "${path.module}/../bin/package.zip"
  source_dir = "${path.module}/../files/"
}

resource "azurerm_storage_blob" "package" {
  depends_on = [null_resource.vault_binary, null_resource.agent_binary]
  
  name = "${sha256(
    join(":", [
      filesha256("${path.module}/../files/host.json"),
      filesha256("${path.module}/../files/function/function.json"),
      filesha256("${path.module}/../files/config/vault.hcl.tpl"),
      filesha256("${path.module}/../files/vault"),
      filesha256("${path.module}/../files/vault-launcher"),
    ])
  )}.zip"
  storage_account_name = azurerm_storage_account.server.name
  storage_container_name = azurerm_storage_container.server.name
  type = "Block"
  source = data.archive_file.server.output_path
}
