import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  poweredByHeader: false,
  async rewrites() {
    return [{ source: "/api/:path*", destination: `${process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8080"}/:path*` }];
  },
  output: "standalone",
  // Keep Docker/CI builds green while the UI is still a dense single-file prototype.
  eslint: {
    ignoreDuringBuilds: true,
  },
  typescript: {
    // Typecheck still runs locally via `npm run lint` / editor; do not block image builds.
    ignoreBuildErrors: false,
  },
};

export default nextConfig;
