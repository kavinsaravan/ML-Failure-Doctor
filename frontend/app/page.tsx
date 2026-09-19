import Link from 'next/link';
import { Zap, Shield, TrendingUp } from 'lucide-react';

export default function Home() {

  return (
    <div className="min-h-screen bg-slate-950">
      {/* Hero Section */}
      <div className="relative bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 overflow-hidden">
        {/* Animated background elements */}
        <div className="absolute inset-0 overflow-hidden">
          <div className="absolute -top-40 -right-40 w-80 h-80 bg-blue-500/10 rounded-full blur-3xl animate-pulse"></div>
          <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-purple-500/10 rounded-full blur-3xl animate-pulse delay-1000"></div>
        </div>

        <div className="container mx-auto px-4 py-20 relative">
          <div className="text-center max-w-6xl mx-auto">
            <div className="inline-block mb-4 px-4 py-2 bg-blue-500/10 border border-blue-500/20 rounded-full">
              <span className="text-blue-400 text-sm font-semibold">AI-Powered Reliability Platform</span>
            </div>
            <h1 className="text-6xl md:text-7xl font-bold text-white mb-6 leading-tight">
              Diagnose failed GPU workloads in{' '}
              <span className="bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent">
                seconds
              </span>
            </h1>
            <p className="text-xl text-slate-300 mb-12 leading-relaxed max-w-3xl mx-auto">
              CrashLens monitors GPU workloads, captures logs and metrics, and uses AI to generate root-cause analysis and repair recommendations.
            </p>
          </div>

          {/* Get Started Button */}
          <div className="flex justify-center mt-12 mb-12">
            <Link
              href="/dashboard"
              className="group px-10 py-4 bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 text-white rounded-xl font-semibold transition-all duration-300 shadow-lg shadow-blue-500/20 hover:shadow-xl hover:shadow-blue-500/30 hover:scale-105 inline-flex items-center gap-2"
            >
              Get Started
              <svg className="w-5 h-5 group-hover:translate-x-1 transition-transform duration-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
              </svg>
            </Link>
          </div>

          {/* Features */}
          <div className="grid md:grid-cols-3 gap-8 max-w-6xl mx-auto">
            <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-blue-500/50 rounded-xl p-6 transition-all duration-300 hover:shadow-xl hover:shadow-blue-500/10 hover:-translate-y-1">
              <div className="w-14 h-14 bg-gradient-to-br from-blue-500 to-blue-600 rounded-xl flex items-center justify-center mb-4 group-hover:scale-110 transition-transform duration-300">
                <Zap className="w-7 h-7 text-white" />
              </div>
              <h3 className="text-xl font-semibold text-white mb-3">
                Instant Diagnosis
              </h3>
              <p className="text-slate-400 leading-relaxed">
                AI-powered failure classification with confidence scoring and evidence extraction from logs.
              </p>
            </div>

            <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-green-500/50 rounded-xl p-6 transition-all duration-300 hover:shadow-xl hover:shadow-green-500/10 hover:-translate-y-1">
              <div className="w-14 h-14 bg-gradient-to-br from-green-500 to-green-600 rounded-xl flex items-center justify-center mb-4 group-hover:scale-110 transition-transform duration-300">
                <Shield className="w-7 h-7 text-white" />
              </div>
              <h3 className="text-xl font-semibold text-white mb-3">
                Universal GPU Support
              </h3>
              <p className="text-slate-400 leading-relaxed">
                Works with NVIDIA (CUDA) and AMD (ROCm) GPUs with automatic detection and platform-specific error diagnosis.
              </p>
            </div>

            <div className="group bg-gradient-to-br from-slate-800 to-slate-800/50 border border-slate-700 hover:border-purple-500/50 rounded-xl p-6 transition-all duration-300 hover:shadow-xl hover:shadow-purple-500/10 hover:-translate-y-1">
              <div className="w-14 h-14 bg-gradient-to-br from-purple-500 to-purple-600 rounded-xl flex items-center justify-center mb-4 group-hover:scale-110 transition-transform duration-300">
                <TrendingUp className="w-7 h-7 text-white" />
              </div>
              <h3 className="text-xl font-semibold text-white mb-3">
                Cost Tracking
              </h3>
              <p className="text-slate-400 leading-relaxed">
                Monitor wasted GPU-seconds on failed jobs and optimize resource usage.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
