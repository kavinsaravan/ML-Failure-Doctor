import Link from 'next/link';
import { ArrowRight, Zap, Shield, TrendingUp } from 'lucide-react';

export default function Home() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900">
      {/* Hero Section */}
      <div className="container mx-auto px-4 py-20">
        <div className="text-center max-w-4xl mx-auto">
          <h1 className="text-6xl font-bold text-white mb-6">
            Diagnose failed ML jobs and AI agents in seconds.
          </h1>
          <p className="text-xl text-slate-300 mb-12 leading-relaxed">
            CrashLens monitors AMD GPU workloads, captures logs and metrics, and uses Gemma to generate root-cause analysis and fix recommendations.
          </p>
          <div className="flex gap-4 justify-center">
            <Link
              href="/dashboard"
              className="px-8 py-4 bg-blue-600 hover:bg-blue-700 text-white rounded-lg font-semibold flex items-center gap-2 transition-colors"
            >
              Go to Dashboard
              <ArrowRight className="w-5 h-5" />
            </Link>
            <a
              href="https://github.com/kavinsaravan/ML-Failure-Doctor"
              target="_blank"
              rel="noopener noreferrer"
              className="px-8 py-4 bg-slate-700 hover:bg-slate-600 text-white rounded-lg font-semibold transition-colors"
            >
              View on GitHub
            </a>
          </div>
        </div>

        {/* Features */}
        <div className="grid md:grid-cols-3 gap-8 mt-20 max-w-6xl mx-auto">
          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <div className="w-12 h-12 bg-blue-600 rounded-lg flex items-center justify-center mb-4">
              <Zap className="w-6 h-6 text-white" />
            </div>
            <h3 className="text-xl font-semibold text-white mb-2">
              Instant Diagnosis
            </h3>
            <p className="text-slate-400">
              AI-powered failure classification with confidence scoring and evidence extraction from logs.
            </p>
          </div>

          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <div className="w-12 h-12 bg-green-600 rounded-lg flex items-center justify-center mb-4">
              <Shield className="w-6 h-6 text-white" />
            </div>
            <h3 className="text-xl font-semibold text-white mb-2">
              AMD ROCm Native
            </h3>
            <p className="text-slate-400">
              Built for AMD GPUs with HIP/ROCm error detection and rocm-smi metrics integration.
            </p>
          </div>

          <div className="bg-slate-800 border border-slate-700 rounded-lg p-6">
            <div className="w-12 h-12 bg-purple-600 rounded-lg flex items-center justify-center mb-4">
              <TrendingUp className="w-6 h-6 text-white" />
            </div>
            <h3 className="text-xl font-semibold text-white mb-2">
              Cost Tracking
            </h3>
            <p className="text-slate-400">
              Monitor wasted GPU-seconds on failed jobs and optimize resource usage.
            </p>
          </div>
        </div>

        {/* Supported Failure Types */}
        <div className="mt-20 max-w-4xl mx-auto">
          <h2 className="text-3xl font-bold text-white text-center mb-8">
            Automatically Detects
          </h2>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            {[
              'GPU Out of Memory',
              'Missing Checkpoint',
              'Dependency Errors',
              'Data Path Errors',
              'Job Timeouts',
              'ROCm Driver Issues',
            ].map((item) => (
              <div
                key={item}
                className="bg-slate-800 border border-slate-700 rounded-lg px-4 py-3 text-slate-300 text-center"
              >
                {item}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
