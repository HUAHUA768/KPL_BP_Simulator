#!/usr/bin/env python3
"""
Download all hero icons from the official 王者荣耀 CDN.
Source: https://pvp.qq.com/web201605/js/herolist.json
Icon URL pattern: https://game.gtimg.cn/images/yxzj/img201606/heroimg/{ename}/{ename}.jpg
"""

import json
import os
import re
import sys
import urllib.request
import time

# Paths
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_DIR = os.path.dirname(SCRIPT_DIR)
HERO_LIST_JSON = os.path.join(PROJECT_DIR, "herolist.json")
SEED_SQL = os.path.join(PROJECT_DIR, "backend", "migrations", "002_seed_data.sql")
OUTPUT_DIR = os.path.join(PROJECT_DIR, "backend", "static", "images", "heroes")

ICON_URL_TEMPLATE = "https://game.gtimg.cn/images/yxzj/img201606/heroimg/{ename}/{ename}.jpg"


def load_official_hero_list():
    """Load the official hero list from herolist.json."""
    with open(HERO_LIST_JSON, "r", encoding="utf-8") as f:
        heroes = json.load(f)
    # Build name -> ename mapping
    name_to_ename = {}
    for h in heroes:
        name_to_ename[h["cname"]] = h["ename"]
    return name_to_ename


def extract_project_heroes():
    """Extract hero names and target filenames from the seed SQL."""
    with open(SEED_SQL, "r", encoding="utf-8") as f:
        content = f.read()

    pattern = r"'([^']+)', '/images/heroes/([^']+)'"
    matches = re.findall(pattern, content)
    return matches  # list of (name, filename)


def build_download_list(name_to_ename, project_heroes):
    """
    Build the download list:
    - Map project hero names to official ename IDs
    - Handle special cases (元流之子 variants use different separator)
    Returns: list of (url, filename, hero_name) and list of not_found names
    """
    download_list = []
    not_found = []

    # Special mapping for 元流之子 variants
    yuanliu_mapping = {
        "元流之子·坦克": "元流之子(坦克)",
        "元流之子·法师": "元流之子(法师)",
        "元流之子·射手": "元流之子(射手)",
        "元流之子·刺客": "元流之子(刺客)",
        "元流之子·辅助": "元流之子(辅助)",
    }

    for name, filename in project_heroes:
        # Try direct match first
        ename = name_to_ename.get(name)

        # Try special mapping
        if ename is None and name in yuanliu_mapping:
            official_name = yuanliu_mapping[name]
            ename = name_to_ename.get(official_name)

        if ename:
            url = ICON_URL_TEMPLATE.format(ename=ename)
            download_list.append((url, filename, name))
        else:
            not_found.append(name)

    return download_list, not_found


def download_icon(url, filepath, hero_name, retries=3):
    """Download a single icon with retries."""
    for attempt in range(retries):
        try:
            req = urllib.request.Request(url, headers={
                "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
                "Referer": "https://pvp.qq.com/",
            })
            with urllib.request.urlopen(req, timeout=30) as resp:
                if resp.status == 200:
                    data = resp.read()
                    with open(filepath, "wb") as f:
                        f.write(data)
                    return True, len(data)
                else:
                    if attempt < retries - 1:
                        time.sleep(1)
                    continue
        except Exception as e:
            if attempt < retries - 1:
                time.sleep(1)
            else:
                return False, str(e)
    return False, "unknown error"


def main():
    print("=" * 60)
    print("KPL BP Simulator - 英雄图标批量下载")
    print("=" * 60)

    # Load data
    print("\n[1/4] 加载官方英雄列表...")
    name_to_ename = load_official_hero_list()
    print(f"  已加载 {len(name_to_ename)} 个英雄")

    print("\n[2/4] 提取项目英雄列表...")
    project_heroes = extract_project_heroes()
    print(f"  已提取 {len(project_heroes)} 个英雄")

    print("\n[3/4] 构建下载列表...")
    download_list, not_found = build_download_list(name_to_ename, project_heroes)
    print(f"  可下载: {len(download_list)} 个")
    print(f"  未找到: {len(not_found)} 个")

    if not_found:
        print("\n  ⚠ 未找到的英雄:")
        for name in not_found:
            print(f"    - {name}")

    # Create output directory
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    # Download
    print(f"\n[4/4] 开始下载到 {OUTPUT_DIR}...")
    success_count = 0
    fail_count = 0
    total_size = 0

    for i, (url, filename, hero_name) in enumerate(download_list, 1):
        filepath = os.path.join(OUTPUT_DIR, filename)
        print(f"  [{i:3d}/{len(download_list)}] {hero_name} -> {filename} ... ", end="", flush=True)

        ok, result = download_icon(url, filepath, hero_name)
        if ok:
            size_kb = result / 1024
            total_size += result
            success_count += 1
            print(f"✓ ({size_kb:.1f} KB)")
        else:
            fail_count += 1
            print(f"✗ ({result})")

        # Small delay to avoid rate limiting
        time.sleep(0.1)

    # Summary
    print("\n" + "=" * 60)
    print("下载完成!")
    print(f"  成功: {success_count} 个")
    print(f"  失败: {fail_count} 个")
    print(f"  总大小: {total_size / 1024:.1f} KB ({total_size / 1024 / 1024:.2f} MB)")
    print(f"  输出目录: {OUTPUT_DIR}")
    print("=" * 60)

    if fail_count > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()