# Examples

1. ["Hello World"](./01_helloworld/hw.go) just prompts LLM and return reply.
2. ["Rainbow"](./02_rainbow/rainbow.go) agent with custom reasoner and codec that handler JSON data.
3. ["Script"](./03_script/) agent with custom reasoner and codec that uses `bash` to manupulate files on the local system.
4. ["Worker"](./04_worker/) is like a "Script" but uses built-in worker profile for same purpose.
5. ["Chain"](./05_chain/) chaining multiple agents togthere.
6. ["Text Processor"](./06_text_processor/processor.go) chaining multiple computational units (agents, functions, etc) to process files.
