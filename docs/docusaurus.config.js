// @ts-check
import {themes as prismThemes} from 'prism-react-renderer';

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'Sloth Kubernetes',
  tagline: 'Multi-cloud Kubernetes cluster provisioning with WireGuard mesh networking',
  favicon: 'img/favicon.ico',

  url: 'https://chalkan3.github.io',
  baseUrl: '/sloth-kubernetes/',

  organizationName: 'chalkan3',
  projectName: 'sloth-kubernetes',

  onBrokenLinks: 'warn',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          path: '.',
          sidebarPath: './sidebars.js',
          editUrl: 'https://github.com/chalkan3/sloth-kubernetes/tree/main/docs/',
          routeBasePath: '/',
          exclude: ['**/node_modules/**', 'src/**', 'static/**', 'package.json', 'package-lock.json', 'docusaurus.config.js', 'sidebars.js', 'README.md', '*.yml', '*.yaml'],
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      image: 'img/social-card.png',
      navbar: {
        title: 'Sloth Kubernetes',
        logo: {
          alt: 'Sloth Kubernetes Logo',
          src: 'img/logo.svg',
        },
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'docs',
            position: 'left',
            label: 'Documentation',
          },
          {
            href: 'https://github.com/chalkan3/sloth-kubernetes',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              {
                label: 'Getting Started',
                to: '/getting-started/installation',
              },
              {
                label: 'User Guide',
                to: '/user-guide/cli-reference',
              },
            ],
          },
          {
            title: 'More',
            items: [
              {
                label: 'GitHub',
                href: 'https://github.com/chalkan3/sloth-kubernetes',
              },
              {
                label: 'Releases',
                href: 'https://github.com/chalkan3/sloth-kubernetes/releases',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} Sloth Kubernetes. Built with Docusaurus.`,
      },
      prism: {
        theme: prismThemes.github,
        darkTheme: prismThemes.dracula,
        additionalLanguages: ['bash', 'yaml', 'json', 'go', 'lisp'],
      },
    }),
};

export default config;
