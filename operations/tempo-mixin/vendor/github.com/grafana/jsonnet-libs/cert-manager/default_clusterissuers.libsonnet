{

  withDefaultIssuer(name, kind='ClusterIssuer', group='cert-manager.io'):: {
    values+:: {
      ingressShim: {
        defaultIssuerName: name,
        defaultIssuerKind: kind,
        defaultIssuerGroup: group,
      },
    },
  },

  clusterIssuer:: {
    new(name): {
      apiVersion: 'cert-manager.io/v1',
      kind: 'ClusterIssuer',
      metadata: {
        name: name,
      },
    },
    withACME(email, server='https://acme-v02.api.letsencrypt.org/directory'): {
      local name = super.metadata.name,
      spec+: {
        acme: {
          // You must replace this email address with your own.
          // Let's Encrypt will use this to contact you about expiring
          // certificates, and issues related to your account.
          email: email,
          server: server,
          privateKeySecretRef: {
            // Secret resource used to store the account's private key.
            name: '%s-account' % name,
          },
        },
      },
    },
    reuseAccount(secret_name): {
      spec+: {
        acme+: {
          // re-use an existing account
          // https://cert-manager.io/docs/configuration/acme/#reusing-an-acme-account
          disableAccountKeyGeneration: true,
          privateKeySecretRef: {
            // Secret resource used to retrieve the account's private key.
            name: secret_name,
          },
        },
      },
    },
    withACMESolverHttp01(class='nginx'): {
      spec+: {
        acme+: {
          // Add a single challenge solver, HTTP01 using nginx
          solvers: [
            {
              http01: {
                ingress: {
                  class: class,
                },
              },
            },
          ],
        },
      },
    },
  },

  // backward compat
  values+:: {
    ingressShim:
      {
        defaultIssuerKind: 'ClusterIssuer',
      }

      + (
        if $._config.default_issuer != null
        then { defaultIssuerName: $._config.default_issuer }
        else {}
      )
      + (
        if $._config.default_issuer_group != null
        then { defaultIssuerGroup: $._config.default_issuer_group }
        else {}
      ),
  },

  // backward compat
  cluster_issuer_staging:
    self.clusterIssuer.new('letsencrypt-staging')
    + self.clusterIssuer.withACME($._config.issuer_email, 'https://acme-staging-v02.api.letsencrypt.org/directory')
    + self.clusterIssuer.withACMESolverHttp01(),

  // backward compat
  cluster_issuer_prod:
    self.clusterIssuer.new('letsencrypt-prod')
    + self.clusterIssuer.withACME($._config.issuer_email)
    + self.clusterIssuer.withACMESolverHttp01(),
}
