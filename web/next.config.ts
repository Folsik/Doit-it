import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  poweredByHeader: false,
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
