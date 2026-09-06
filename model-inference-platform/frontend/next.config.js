/** @type {import('next').NextConfig} */
const INFERENCE  = process.env.NEXT_PUBLIC_API_URL      || 'http://localhost:8080';
const MANAGEMENT = process.env.NEXT_PUBLIC_MGMT_API_URL || 'http://localhost:8081';

const nextConfig = {
  output: 'standalone',
  async rewrites() {
    return [
      { source: '/api/:path*',    destination: `${INFERENCE}/api/:path*` },
      { source: '/v0/:path*',     destination: `${MANAGEMENT}/v0/:path*` },
      { source: '/v1/:path*',     destination: `${INFERENCE}/v1/:path*` },
      { source: '/health/:path*', destination: `${INFERENCE}/health/:path*` },
      { source: '/metrics',       destination: `${INFERENCE}/metrics` },
    ];
  },
}
module.exports = nextConfig
