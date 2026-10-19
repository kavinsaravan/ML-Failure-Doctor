import type { Metadata } from "next";
import "./globals.css";
import AccessGate from "@/components/access-gate";

export const metadata: Metadata = {
  title: "CrashLens - AI-Powered GPU Workload Failure Diagnosis",
  description: "Track GPU workload logs and metrics with private workspaces and optional Fireworks diagnosis for CUDA and ROCm.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className="h-full antialiased"
    >
      <body className="min-h-full flex flex-col"><AccessGate>{children}</AccessGate></body>
    </html>
  );
}
