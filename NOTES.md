# Build Log: Touch Grass Quest

## Step 0: Vision Feasibility Test
I wrote a small test tool (`cmd/visiontest`) to verify if the LLM provider supports image uploads via the OpenAI-compatible `/chat/completions` endpoint. Initially, some open-weight text models might ignore image inputs. I proposed adding a `VISION_MODEL` environment variable specifically to route image queries to a capable multimodal model (e.g. `LLM_MODEL=llama-3` for text and `VISION_MODEL=llava-1.5` for vision), preventing silent failures. Since my test environment lacked an API key during the agent run, I proceeded with a mocked fake verifier and built the rest of the application robustly so it works once configured.

## Prompt Iteration and Hallucinations
When designing the prompt for the vision model, a common failure mode is the model "hallucinating" elements that aren't there, or being too generous with the "outdoors" requirement (e.g., accepting a photo of a laptop screen showing a forest).

**The fix:** I specifically instructed the model: "Be fair but honest, accept reasonable interpretations, do not invent objects. Treat a photo of a screen or printed picture as not outdoors." The logic was moved out of the model and into Go: the model only returns a strict JSON object `{ "match": bool, "outdoors": bool, "reason": string }`. Go handles the final verdict. If it's a match but indoors, we reply gently: "Looks like it's indoors or a screen, take it outside."

## The Evaluator (`cmd/eval`)
I built a live evaluation harness that reads `testdata/eval/manifest.json`. You can place test images in that directory to benchmark your specific model. The harness tests parsing (handling malformed JSON or `<thought>` tags), and strict verdict matching. 

A sample evaluation table output would look like this (results depend on the model used):

| Image | Task | Expected | Actual | Reason | Pass/Fail | Latency |
|-------|------|----------|--------|--------|-----------|---------|
| true_leaf.jpg | t1 | true | true | "Clear shot of an autumn leaf outside." | PASS | 2.1s |
| screen_leaf.jpg | t1 | false | false | "Photo is of a computer screen." | PASS | 1.8s |
| indoor_desk.jpg | t2 | false | false | "Taken indoors on a desk." | PASS | 1.9s |

## Latency
The frontend downscales images to a maximum of 1024px and encodes them as ~80% quality JPEG before sending them to the backend, drastically reducing upload time. Latency will be dominated by the chosen vision model's TTFT (Time To First Token) and processing speed. The evaluator will print median and slowest latencies, which typically hover around 2-4 seconds for modern open-weight models on fast providers.

## Imperfections & Cheating
- **Screen Spoofing**: Highly realistic 4K screens or printed photos placed outside might still fool the vision model. The prompt instructs it to look for screens, but it's not foolproof.
- **JSON Output**: If the model ignores the strict JSON instruction and writes conversational text, we employ a retry mechanism (1 retry) and fallback safely. We also strip `<thought>` tags if the model is a reasoning model.
