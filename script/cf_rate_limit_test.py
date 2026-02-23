import concurrent.futures
import json
import time
import urllib.error
import urllib.request


API_URL = "https://codeforces.com/api/user.status?handle=Xiao_Shuai_Ge&from=1&count=10"
TEST_REQUEST_COUNT = 10
TEST_GAP_SECONDS = 5
INITIAL_INTERVALS = [2.1, 1.1, 0.6, 0.3]
MIN_INTERVAL = 0.1
INTERVAL_STEP = 0.1
TIMEOUT_SECONDS = 10


def request_once(planned_start):
    request = urllib.request.Request(
        API_URL,
        headers={
            "User-Agent": "ACK-ACM-GAME-CF-Rate-Test/1.0"
        },
    )
    actual_start = time.perf_counter()
    try:
        with urllib.request.urlopen(request, timeout=TIMEOUT_SECONDS) as response:
            raw = response.read()
        end = time.perf_counter()
        elapsed = end - actual_start
        text = raw.decode("utf-8")
        data = json.loads(text)
        if data.get("status") == "OK":
            return True, elapsed, "", planned_start, actual_start, end, text
        comment = data.get("comment") or "API返回非OK"
        return False, elapsed, comment, planned_start, actual_start, end, text
    except urllib.error.HTTPError as exc:
        end = time.perf_counter()
        elapsed = end - actual_start
        return False, elapsed, f"HTTPError {exc.code}", planned_start, actual_start, end, ""
    except Exception as exc:
        end = time.perf_counter()
        elapsed = end - actual_start
        return False, elapsed, f"异常 {exc}", planned_start, actual_start, end, ""


def run_test(interval):
    success_count = 0
    total_elapsed = 0.0
    errors = []
    results = []
    base = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=TEST_REQUEST_COUNT) as executor:
        futures = []
        for idx in range(TEST_REQUEST_COUNT):
            planned_start = base + idx * interval
            now = time.perf_counter()
            sleep_seconds = planned_start - now
            if sleep_seconds > 0:
                time.sleep(sleep_seconds)
            future = executor.submit(request_once, planned_start)
            futures.append((idx + 1, planned_start, future))
        for idx, planned_start, future in futures:
            ok, elapsed, error, planned, actual, end, text = future.result()
            results.append((idx, ok, elapsed, error, planned, actual, end, text))
            total_elapsed += elapsed
            if ok:
                success_count += 1
            else:
                errors.append(error)
    for idx, ok, elapsed, error, planned, actual, end, text in results:
        drift = actual - planned
        status = "OK" if ok else f"失败 {error}"
        print(
            f"请求#{idx:02d} 计划+{planned - base:.3f}s 实际+{actual - base:.3f}s "
            f"偏差 {drift:+.3f}s 耗时 {elapsed:.3f}s {status}"
        )
        if text:
            print(f"响应内容：{text}")
    avg_elapsed = total_elapsed / TEST_REQUEST_COUNT
    failed = success_count != TEST_REQUEST_COUNT
    return failed, success_count, avg_elapsed, errors


def run_series(intervals):
    last_success_interval = None
    for interval in intervals:
        print(f"开始测试，固定间隔 {interval:.1f}s，每次 {TEST_REQUEST_COUNT} 请求")
        failed, success_count, avg_elapsed, errors = run_test(interval)
        print(f"结果：成功 {success_count}/{TEST_REQUEST_COUNT}，平均耗时 {avg_elapsed:.3f}s")
        if failed:
            unique_errors = list(dict.fromkeys(errors))
            print(f"失败原因：{'; '.join(unique_errors)}")
        else:
            last_success_interval = interval
        print(f"等待 {TEST_GAP_SECONDS}s 后进行下一轮")
        time.sleep(TEST_GAP_SECONDS)
    return last_success_interval


def run_until_fail(start_interval):
    last_success_interval = None
    interval = start_interval
    while interval >= MIN_INTERVAL:
        print(f"开始测试，固定间隔 {interval:.1f}s，每次 {TEST_REQUEST_COUNT} 请求")
        failed, success_count, avg_elapsed, errors = run_test(interval)
        print(f"结果：成功 {success_count}/{TEST_REQUEST_COUNT}，平均耗时 {avg_elapsed:.3f}s")
        if failed:
            unique_errors = list(dict.fromkeys(errors))
            print(f"失败原因：{'; '.join(unique_errors)}")
            return last_success_interval, interval
        last_success_interval = interval
        print(f"等待 {TEST_GAP_SECONDS}s 后进行下一轮")
        time.sleep(TEST_GAP_SECONDS)
        interval = round(interval - INTERVAL_STEP, 1)
    return last_success_interval, None


def main():
    print("开始测试 Codeforces 公共 API 频率限制")
    last_success_interval = run_series(INITIAL_INTERVALS)
    if last_success_interval is None:
        print("基础测试阶段无成功间隔，停止继续降低间隔")
        return
    next_interval = round(last_success_interval - INTERVAL_STEP, 1)
    if next_interval < MIN_INTERVAL:
        print(f"已达到最小间隔 {last_success_interval:.1f}s")
        return
    print(f"开始继续降低间隔，从 {next_interval:.1f}s 起")
    success_interval, failed_interval = run_until_fail(next_interval)
    if failed_interval is None:
        print(f"在 {MIN_INTERVAL:.1f}s 仍然通过，未触发失败")
        return
    if success_interval is None:
        print(f"在 {failed_interval:.1f}s 已失败，未找到可用更小间隔")
        return
    print(f"最短稳定间隔约为 {success_interval:.1f}s，失败发生在 {failed_interval:.1f}s")


if __name__ == "__main__":
    main()
