default_max_request_duration = "90s"
disable_clustering           = true
disable_mlock                = true
ui                           = true

log_format = "json"
log_level  = "Info"

listener "tcp" {
  address         = "[::]:{{ env "FUNCTIONS_CUSTOMHANDLER_PORT" }}"
  tls_disable = "true"
  x_forwarded_for_authorized_addrs = ["127.0.0.1"]
}

seal "azurekeyvault" {
    tenant_id = "{{ env "AZURE_TENANT_ID" }}"
    vault_name = "{{ env "AZURE_KEYVAULT_VAULT_NAME" }}"
    key_name = "vault-unseal"
}

storage "azure" {
    accountName = "{{ env "AZURE_ACCOUNT_NAME" }}"
    accountKey  = "{{ env "AZURE_STORAGE_KEY" }}"
    container   = "{{ env "AZURE_BLOB_CONTAINER" }}"
}
