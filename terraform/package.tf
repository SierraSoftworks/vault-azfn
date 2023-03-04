resource "null_resource" "vault_binary" {
    triggers = {
        every_run = timestamp()
    }

    provisioner "local-exec" {
        command = <<EOH
set -e
curl -sSL -o vault.zip https://releases.hashicorp.com/vault/${var.vault_version}/vault_${var.vault_version}_linux_amd64.zip
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
curl -sSL -o vault-agent https://github.com/SierraSoftworks/vault-azfn/releases/download/v${var.vault_agent_version}/vault-agent
chmod 0755 vault-agent
mv vault-agent ${path.module}/../files/vault-agent
EOH
    }
}

resource "null_resource" "github_plugin_binary" {
    triggers = {
        every_run = timestamp()
    }

    provisioner "local-exec" {
        command = <<EOH
set -e
curl -sSL -o vault-plugin-secrets-github https://github.com/martinbaillie/vault-plugin-secrets-github/releases/download/v${var.vault_github_plugin_version}/vault-plugin-secrets-github-linux-amd64
chmod 0755 vault-plugin-secrets-github
mkdir -p ${path.module}/../files/plugins
mv vault-plugin-secrets-github ${path.module}/../files/plugins/vault-plugin-secrets-github
EOH
    }
}

data "archive_file" "server" {
  depends_on = [null_resource.vault_binary, null_resource.agent_binary]

  type        = "zip"
  output_path = "${path.module}/files/package.zip"
  source_dir = "${path.module}/../files/"
}

resource "azurerm_storage_blob" "package" {
  name = "${sha256(
    join(":", [
      var.vault_version,
      "agent-${var.vault_agent_version}",
      "plugin-github-${var.vault_github_plugin_version}",
      filesha256("${path.module}/../files/host.json"),
      filesha256("${path.module}/../files/function/function.json"),
      filesha256("${path.module}/../files/config/vault.hcl.tpl"),
    ])
  )}.zip"
  storage_account_name = azurerm_storage_account.server.name
  storage_container_name = azurerm_storage_container.server.name
  type = "Block"
  source = data.archive_file.server.output_path
}

data "azurerm_storage_account_blob_container_sas" "package" {
  connection_string = azurerm_storage_account.server.primary_connection_string
  container_name    = azurerm_storage_container.server.name

  start = timestamp()
  expiry = timeadd(timestamp(), "8000h")

  permissions {
    read   = true
    add    = false
    create = false
    write  = false
    delete = false
    list   = false
  }
}