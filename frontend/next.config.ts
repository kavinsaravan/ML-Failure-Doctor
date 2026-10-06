import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Docker needs a standalone server; other deployments use the default output.
  ...(process.env.NEXT_OUTPUT_STANDALONE === "true" ? { output: "standalone" } : {}),
};

export default nextConfig;
