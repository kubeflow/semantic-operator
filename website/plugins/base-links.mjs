/**
 * Prefix root-relative links with the site's base path.
 *
 * Astro serves this site under a base (`/semantic-operator` on GitHub Pages).
 * Starlight rewrites the links it generates itself, such as the sidebar, but a
 * plain markdown link written as `/start/quickstart` is emitted verbatim and
 * 404s once a base is in play.
 *
 * Authors should keep writing root-relative links, because they read well in
 * source and survive a change of base. This plugin does the rewriting at build
 * time so nobody has to remember.
 */
export function satteriBaseLinks(base) {
  const prefix = base === '/' ? '' : base.replace(/\/$/, '');

  const needsPrefix = (href) =>
    typeof href === 'string' &&
    href.startsWith('/') &&
    !href.startsWith('//') &&
    !href.startsWith(prefix + '/') &&
    href !== prefix;

  return {
    name: 'base-links',
    element: {
      // An empty filter matches every element. Images and other src-bearing
      // elements need the same treatment as anchors.
      filter: [],
      visit(node, ctx) {
        if (!prefix) return;
        const props = node.properties ?? {};
        if (node.tagName === 'a' && needsPrefix(props.href)) {
          ctx.setProperty(node, 'href', prefix + props.href);
        }
        if (needsPrefix(props.src)) {
          ctx.setProperty(node, 'src', prefix + props.src);
        }
      },
    },
  };
}
