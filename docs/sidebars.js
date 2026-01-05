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
      ],
    },
    {
      type: 'category',
      label: 'Architecture',
      items: [
        'architecture/index',
        'advanced/architecture',
      ],
    },
    'faq',
  ],
};

export default sidebars;
