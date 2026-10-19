"""
Example: Track a real PyTorch training job with CrashLens
"""

import sys
sys.path.insert(0, '..')

import torch
import torch.nn as nn
import torch.optim as optim
from crashlens import WorkloadTracker

# Initialize tracker
tracker = WorkloadTracker("https://invigorating-empathy-production-dee5.up.railway.app")

# Define a simple model
class SimpleModel(nn.Module):
    def __init__(self):
        super().__init__()
        self.fc = nn.Linear(10, 1)
    
    def forward(self, x):
        return self.fc(x)

# Track training with context manager
with tracker.track("PyTorch Training Example"):
    print("Initializing model...")
    model = SimpleModel()
    optimizer = optim.Adam(model.parameters())
    criterion = nn.MSELoss()
    
    print("Starting training...")
    for epoch in range(10):
        # Dummy data
        x = torch.randn(32, 10)
        y = torch.randn(32, 1)
        
        optimizer.zero_grad()
        output = model(x)
        loss = criterion(output, y)
        loss.backward()
        optimizer.step()
        
        print(f"Epoch {epoch+1}/10, Loss: {loss.item():.4f}")
    
    print("Training completed successfully!")
