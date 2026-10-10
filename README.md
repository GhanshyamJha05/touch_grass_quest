# 🌿 Touch Grass Quest

**Touch Grass Quest** is a daily, mobile-first web app built for the DEV Challenge. It encourages users to put down their screens and interact with the real world by giving them one unique outdoor photography task every day. 

Upload your photo, and an open-weight Vision AI will determine if you successfully found the item outdoors. Build your streak, and go touch some grass!

## ✨ Features
- **Daily Quests:** A new deterministic, outdoor-focused task generated for you every day.
- **Open-Weight AI Verification:** Verification is powered by open-source vision models via any OpenAI-compatible API.
- **Mobile-First UX:** Sleek, responsive, dark-mode UI designed to feel like a native mobile app.
- **Privacy & Speed:** Image downscaling happens *locally* on your device before upload, saving bandwidth and protecting raw photo data.
- **Streak Tracking:** Tracks your daily success streak locally on your device.

## 🚀 Quick Start

### Prerequisites
- [Go 1.22+](https://go.dev/)
- An API Key from an OpenAI-compatible provider (e.g., Together AI, Fireworks AI, or Gemini).

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/GhanshyamJha05/touch_grass_quest.git
   cd touch_grass_quest
   ```
2. Copy the environment variables template:
   ```bash
   cp .env.example .env
   ```
3. Update your `.env` file with your API credentials.
4. Run the application:
   ```bash
   go run main.go
   ```
5. Open your browser to `http://localhost:8080`.

## 🛠️ Tech Stack
- **Backend:** Go (Standard Library `net/http`)
- **Frontend:** Vanilla HTML, CSS, JavaScript (No external frameworks)
- **AI Verification:** OpenAI-Compatible Vision API (Zero-shot image classification)

## 🧪 Testing & Evaluation
- **Vision Test CLI:** Test your vision model connectivity and prompting directly from the command line:
  `go run cmd/visiontest/main.go testdata/sample.jpg`
- **Offline Evaluation Harness:** Evaluate the AI's accuracy across a dataset of test images against a truth manifest:
  `go run cmd/eval/main.go --dataset path/to/dataset --manifest manifest.json`

## 🐳 Docker Deployment
A multi-stage `Dockerfile` is included for lightweight, production-ready containerization.
```bash
docker build -t touch-grass-quest .
docker run -p 8080:8080 --env-file .env touch-grass-quest
```

---
*Built for the DEV Challenge: Open-Source AI*
