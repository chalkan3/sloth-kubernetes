/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docs: [
    'index',
    {
      type: 'category',
      label: 'Getting Started',
      collapsed: false,
      items: [
        'getting-started/index',
        'getting-started/installation',
        'getting-started/quickstart',
      ],
    },
    {
      type: 'category',
      label: 'Configuration',
      items: [
        'configuration/lisp-format',
        'configuration/builtin-functions',
        'configuration/backend',
        'configuration/examples',
      ],
    },
    {
      type: 'category',
      label: 'User Guide',
      items: [
        'user-guide/index',
        'user-guide/cli-reference',
        'user-guide/stacks',
        'user-guide/salt',
        'user-guide/kubectl',
        'user-guide/upgrade',
        'user-guide/benchmark',
        'user-guide/backup',
        'user-guide/argocd',
        'user-guide/vpn',
        'user-guide/health',
      ],
    },
    'faq',
  ],
};

export default sidebars;
