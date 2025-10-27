# sloth-kubernetes Documentation

This directory contains the source for the sloth-kubernetes documentation website, built with Jekyll and the just-the-docs theme.

## Local Development

To build and preview the documentation locally:

```bash
cd docs
bundle install
bundle exec jekyll serve
```

Then open http://localhost:4000/sloth-kubernetes in your browser.

## Deployment

The documentation is automatically deployed to GitHub Pages via GitHub Actions whenever changes are pushed to the `main` branch.

## Structure

- `index.md` - Homepage
- `getting-started/` - Installation and quick start guides
- `user-guide/` - Comprehensive usage documentation
- `architecture/` - Technical architecture and design
- `cli-reference/` - CLI command reference
- `examples/` - Example configurations and use cases

## Contributing

To contribute to the documentation:

1. Edit the relevant `.md` files
2. Test locally with `bundle exec jekyll serve`
3. Submit a pull request

## Theme

We use the [just-the-docs](https://just-the-docs.com/) Jekyll theme.
