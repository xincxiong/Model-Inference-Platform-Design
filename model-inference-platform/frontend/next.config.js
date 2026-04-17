/** @type {import('next').NextConfig} */
const BACKEND = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

const nextConfig = {
  output: 'standalone',
  async rewrites() {
    return [
      { source: '/api/:path*',    destination: `${BACKEND}/api/:path*` },
      { source: '/v0/:path*',     destination: `${BACKEND}/v0/:path*` },
      { source: '/v1/:path*',     destination: `${BACKEND}/v1/:path*` },
      { source: '/health/:path*', destination: `${BACKEND}/health/:path*` },
      { source: '/metrics',       destination: `${BACKEND}/metrics` },
    ];
  },
}
module.exports = nextConfig
