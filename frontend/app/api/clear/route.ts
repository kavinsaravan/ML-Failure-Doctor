import { NextResponse } from 'next/server';

export async function POST() {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
  const apiKey = process.env.CRASHLENS_API_KEY; // Server-only, never sent to client

  try {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };

    // Only include Authorization header if API key is set
    if (apiKey) {
      headers['Authorization'] = `Bearer ${apiKey}`;
    }

    const response = await fetch(`${apiUrl}/workloads/clear`, {
      method: 'DELETE',
      headers,
    });

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to clear workloads' },
        { status: response.status }
      );
    }

    return NextResponse.json({ success: true });
  } catch (error) {
    console.error('Error clearing workloads:', error);
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    );
  }
}
