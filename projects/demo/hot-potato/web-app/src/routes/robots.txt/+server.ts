import { site } from '../../config/site';

export const prerender = true;

export function GET() {
  return new Response(`User-agent: *\nAllow: /\nSitemap: ${site.origin}/sitemap.xml\n`, {
    headers: { 'Content-Type': 'text/plain; charset=utf-8' }
  });
}
