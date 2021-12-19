# Hashicorp Vault on Azure Functions
Do you love the idea of running Hashicorp's (phenomenal) Vault, but can't
stomach the idea of spending hundreds of dollars per month hosting a cluster?
Yeah, me too - so here's a better idea: let's run it on Azure Functions Consumption Plan,
scale to zero when we don't use it, pay a few cents per month
and still have all the lovely functionality we wanted in the first place.

This idea is unashamedly stolen from [Kelsey Hightower](https://twitter.com/kelseyhightower)'s similar
work to get Vault deployed on Google's [CloudRun](https://github.com/kelseyhightower/serverless-vault-with-cloud-run)
serverless offering. I'll be honest, it was nowhere near as straightforward for Azure Functions
(primarily as a result of Vault not allowing environment variables to be used in your `listener` specs).

To work around this, I have built a lightweight launcher which is responsible for templating
your `vault.hcl` configuration (injecting environment variables into it) and converting Vault's JSON
log messages into rich trace events for AppInsights (because who doesn't love rich trace events?).

All of this is bundled up and deployed using Terraform, which should make
getting started a relatively painless experience. Of course, if you run into
issues, please open an issue and I'll try to help out (keeping in mind that
I do this in my spare time).