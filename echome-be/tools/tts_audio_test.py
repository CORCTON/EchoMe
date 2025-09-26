#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import asyncio
import json
import sys
import time
import base64
import argparse
from websockets import connect
import logging

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# 测试用例
test_cases = [
    {
        "name": "短文本测试(10字符)",
        "text": "你好世界",
        "expect_full_audio": True
    },
    {
        "name": "短文本带标点测试",
        "text": "你好，这是一个测试。",
        "expect_full_audio": True
    },
    {
        "name": "中等长度文本测试",
        "text": "这是一个大约50个字符左右的中等长度文本，用于测试TTS系统对不同长度文本的处理能力。",
        "expect_full_audio": True
    },
    {
        "name": "长文本测试",
        "text": "这是一个较长的文本，用于测试TTS系统的完整性。通过发送一个包含多个句子的长文本，我们可以观察系统是否能够正确地分割文本，并且完整地生成对应的音频。这对于验证我们之前修复的短文本处理问题非常重要。",
        "expect_full_audio": True
    },
    {
        "name": "多标点符号文本测试",
        "text": "你好！这是第一句话。这是第二句话，包含逗号。这是第三句话；包含分号。这是最后一句话？包含问号。",
        "expect_full_audio": True
    },
]

# 测试结果类
class TestResult:
    def __init__(self, test_case):
        self.test_case = test_case
        self.success = False
        self.message = ""
        self.audio_received = False
        self.audio_size = 0
        self.response_time = 0
        self.binary_messages_received = 0

    def to_dict(self):
        return {
            "test_case": self.test_case,
            "success": self.success,
            "message": self.message,
            "audio_received": self.audio_received,
            "audio_size": self.audio_size,
            "binary_messages_received": self.binary_messages_received,
            "response_time": self.response_time
        }

async def run_single_test(test_case, url, character_id, timeout=30):
    """运行单个测试用例"""
    result = TestResult(test_case["name"])
    logger.info(f"开始测试: {test_case['name']}")
    logger.debug(f"测试文本: {test_case['text']}")

    try:
        # 构建URL
        final_url = url
        if character_id:
            if '?' in final_url:
                final_url += f"&characterId={character_id}"
            else:
                final_url += f"?characterId={character_id}"
        
        logger.info(f"连接到: {final_url}")
        
        # 连接WebSocket
        async with connect(final_url, ping_interval=10) as websocket:
            # 等待连接建立消息
            start_time = time.time()
            conn_established = False
            
            # 等待连接建立消息，最多等待5秒
            try:
                conn_message = await asyncio.wait_for(websocket.recv(), timeout=5)
                logger.debug(f"连接建立消息: {conn_message}")
                conn_established = True
            except asyncio.TimeoutError:
                result.message = "等待连接建立超时"
                return result

            if not conn_established:
                result.message = "未收到连接建立消息"
                return result

            # 发送测试消息
            test_message = {
                "text": test_case["text"],
                "stream": True
            }
            await websocket.send(json.dumps(test_message))
            logger.info(f"已发送测试消息，长度: {len(test_case['text'])}字符")

            # 重置开始时间，只计算响应时间
            start_time = time.time()

            # 收集音频数据和处理响应
            has_received_audio = False
            audio_size = 0
            binary_messages = 0
            stream_end_received = False
            
            # 开始接收消息，直到收到stream_end或超时
            while time.time() - start_time < timeout:
                try:
                    # 尝试接收消息，设置短暂超时以便检查是否超时
                    try:
                        message = await asyncio.wait_for(websocket.recv(), timeout=0.1)
                    except asyncio.TimeoutError:
                        continue

                    # 处理二进制消息（音频数据）
                    if isinstance(message, bytes):
                        binary_messages += 1
                        audio_size += len(message)
                        has_received_audio = True
                        logger.debug(f"收到音频数据，包大小: {len(message)} bytes，总大小: {audio_size} bytes")
                        continue

                    # 处理文本消息
                    try:
                        data = json.loads(message)
                        msg_type = data.get("type", "")
                        
                        if msg_type == "stream_chunk":
                            content = data.get("content", "")
                            logger.debug(f"收到文本块，长度: {len(content)}字符")
                        elif msg_type == "stream_end":
                            stream_end_received = True
                            response_time = int((time.time() - start_time) * 1000)  # 毫秒
                            logger.info(f"收到流结束消息，响应时间: {response_time}ms")
                            break
                        elif msg_type == "error":
                            error_msg = data.get("message", "未知错误")
                            result.message = f"服务器返回错误: {error_msg}"
                            return result
                    except json.JSONDecodeError:
                        logger.warning(f"无法解析消息: {message}")
                        continue

                except Exception as e:
                    logger.error(f"接收消息时出错: {e}")
                    result.message = f"接收消息时出错: {str(e)}"
                    return result

            # 计算响应时间
            response_time = int((time.time() - start_time) * 1000)  # 毫秒

            # 验证结果
            if not stream_end_received:
                result.message = "未收到流结束消息，测试超时"
                return result

            if not has_received_audio and test_case["expect_full_audio"]:
                result.message = "未收到任何音频数据"
                return result

            if audio_size == 0 and test_case["expect_full_audio"]:
                result.message = "收到的音频数据为空"
                return result

            # 测试成功
            result.success = True
            result.audio_received = has_received_audio
            result.audio_size = audio_size
            result.binary_messages_received = binary_messages
            result.response_time = response_time
            result.message = f"测试成功，音频大小: {audio_size} bytes，二进制消息数: {binary_messages}，响应时间: {response_time} ms"
            logger.info(result.message)

    except Exception as e:
        logger.error(f"测试执行出错: {e}")
        result.message = f"测试执行出错: {str(e)}"

    return result

async def main():
    # 解析命令行参数
    parser = argparse.ArgumentParser(description='TTS音频完整性测试工具')
    parser.add_argument('--url', type=str, default='ws://localhost:8081/ws/voice-conversation', help='WebSocket服务器URL')
    parser.add_argument('--characterId', type=str, default='', help='角色ID')
    parser.add_argument('--output', type=str, default='tts_test_results.json', help='测试结果输出文件')
    parser.add_argument('--debug', action='store_true', help='启用调试日志')
    parser.add_argument('--timeout', type=int, default=30, help='测试超时时间(秒)')
    args = parser.parse_args()

    # 如果启用调试模式，设置日志级别为DEBUG
    if args.debug:
        logger.setLevel(logging.DEBUG)

    results = []
    success_count = 0

    for tc in test_cases:
        # 运行单个测试
        result = await run_single_test(tc, args.url, args.characterId, args.timeout)
        results.append(result.to_dict())

        # 打印测试结果
        status = "✅ 成功" if result.success else "❌ 失败"
        print(f"测试: {tc['name']} - {status}")
        if not result.success:
            print(f"  错误: {result.message}")
        
        if result.success:
            success_count += 1

        # 测试间隔
        await asyncio.sleep(2)

    # 保存测试结果到文件
    with open(args.output, 'w', encoding='utf-8') as f:
        json.dump(results, f, ensure_ascii=False, indent=2)

    # 打印总结
    print("\n=== 测试总结 ===")
    print(f"总测试用例: {len(test_cases)}")
    print(f"成功: {success_count}")
    print(f"失败: {len(test_cases) - success_count}")
    print(f"成功率: {success_count / len(test_cases) * 100:.1f}%")
    print(f"测试结果已保存至: {args.output}")

if __name__ == '__main__':
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\n测试被用户中断")
        sys.exit(0)
    except Exception as e:
        print(f"程序执行出错: {e}")
        sys.exit(1)