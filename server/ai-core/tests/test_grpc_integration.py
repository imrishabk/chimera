import sys
import asyncio

import grpc

sys.path.insert(0, "grpc_gen/generated")
from ai_core.v1 import ai_core_pb2, ai_core_pb2_grpc


async def test_health():
    async with grpc.aio.insecure_channel("localhost:50051") as channel:
        stub = ai_core_pb2_grpc.AIServiceStub(channel)
        response = await stub.Health(ai_core_pb2.HealthRequest())
        print(f"✅ Health: Status={response.status}, version={response.version}")
        assert response.status == "healthy"
        return True


async def test_chat():
    async with grpc.aio.insecure_channel("localhost:50051") as channel:
        stub = ai_core_pb2_grpc.AIServiceStub(channel)
        request = ai_core_pb2.ChatRequest(
            session_id="fddec2e7-27c9-447b-9266-174dd765a4c3",
            messages=[ai_core_pb2.Message(role="user", content="Say hello briefly")],
        )
        response = await stub.Chat(request)
        print(f"✅ Chat: {response.message.content[:50]}...")
        assert response.done is True
        assert response.message.role == "assistant"
        return True


async def test_chat_stream():
    async with grpc.aio.insecure_channel("localhost:50051") as channel:
        stub = ai_core_pb2_grpc.AIServiceStub(channel)
        request = ai_core_pb2.ChatRequest(
            session_id="d47de34d-aebc-4c7c-881a-2c5731b09115",
            messages=[ai_core_pb2.Message(role="user", content="Count to 3")],
        )
        chunks = []
        async for response in stub.ChatStream(request):
            chunks.append(response)
            if response.done:
                break
        print(f"✅ Chatstream: {len(chunks)} chunks received")
        assert chunks[-1].done is True


async def test_ingest():
    async with grpc.aio.insecure_channel("localhost:50051") as channel:
        stub = ai_core_pb2_grpc.AIServiceStub(channel)
        request = ai_core_pb2.IngestRequest(
            session_id="fd81c9ad-c265-408c-9cb0-32a808945eed",
            documents=[
                ai_core_pb2.Document(
                    content="Machine learning is a subset of AI that enables systems to learn from data.",
                    source="ml_intro.txt",
                    source_type="text",
                    doc_id="doc-ml-1",
                    chunk_index=0,
                    page_title="ML Basics",
                    chapter="Chapter 1",
                )
            ],
        )
        response = await stub.IngestDocuments(request)
        print(
            f"✅ IngestDocuments: success={response.success}, count={response.count}, ids={response.docs_id}"
        )
        assert response.success is True
        assert response.count > 0
        return response.docs_id


async def test_query(docs_id):
    async with grpc.aio.insecure_channel("localhost:50051") as channel:
        stub = ai_core_pb2_grpc.AIServiceStub(channel)
        request = ai_core_pb2.QueryRequest(
            session_id="fd81c9ad-c265-408c-9cb0-32a808945eed",
            query="What is machine learning?",
            k=4,
        )
        response = await stub.QueryRAG(request)
        print(
            f"✅ QueryRAG: answer_len={len(response.answer)}, sources={len(response.sources)}"
        )
        assert len(response.answer) > 0
        assert len(response.sources) > 0
        assert "[Source" in response.answer
        return True


async def main():
    print("Running gRPC integration tests...\n")

    await test_health()
    await test_chat()
    await test_chat_stream()
    docs_id = await test_ingest()
    await test_query(docs_id)

    print("\n✅ All tests passed!")


if __name__ == "__main__":
    asyncio.run(main())
