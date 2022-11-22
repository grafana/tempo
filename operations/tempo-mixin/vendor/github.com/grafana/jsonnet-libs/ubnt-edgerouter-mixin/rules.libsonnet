{
  prometheusRules+:: {
    groups+: [
      {
        name: 'ubnt.rules',
        rules: [
          {
            record: 'ifNiceName',
            expr: 'label_join(ifAdminStatus,"nicename", ":", "ifName", "ifAlias")',
          },
        ],
      },
    ],
  },
}
