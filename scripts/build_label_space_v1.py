#!/usr/bin/env python3
"""
构建结构化中文标签库 v1
来源：Open Images 600 标签翻译 + 现有 55 个标签
输出：config/label_space_v1.json
"""
import json

existing_labels = [
    "城市街景", "古建筑", "现代建筑", "公园园林", "海滨沙滩",
    "山川湖泊", "机场车站", "室内场景", "夜景灯光", "道路桥梁",
    "校园场景", "商业街市", "历史文化街区", "自然风光", "城市天际线",
    "花朵植物", "树木森林", "天空云彩", "水面倒影", "交通工具",
    "飞机航空", "动物宠物", "人物合影", "儿童", "演出舞台",
    "会议讲座", "美食餐饮", "体育运动", "园林景观", "喷泉雕塑",
    "蓝天白云", "日出日落", "雪景冰霜", "雨雾天气", "逆光剪影",
    "微距特写", "广角视野", "对称构图", "倒影镜面", "红墙黄瓦",
    "霓虹灯光", "烟火表演", "人群聚集", "书法艺术", "国旗旗帜",
    "宫殿建筑", "庙宇祠堂", "塔楼钟楼", "城墙城门", "亭台楼阁",
    "码头港口", "摩天轮", "高楼大厦", "别墅住宅", "农田乡村",
]

existing_map = {
    "城市街景": "scene", "古建筑": "scene", "现代建筑": "scene", "公园园林": "scene", "海滨沙滩": "scene",
    "山川湖泊": "scene", "机场车站": "scene", "室内场景": "scene", "夜景灯光": "attribute", "道路桥梁": "scene",
    "校园场景": "scene", "商业街市": "scene", "历史文化街区": "scene", "自然风光": "scene", "城市天际线": "scene",
    "花朵植物": "object", "树木森林": "scene", "天空云彩": "scene", "水面倒影": "attribute", "交通工具": "object",
    "飞机航空": "object", "动物宠物": "object", "人物合影": "person", "儿童": "person", "演出舞台": "scene",
    "会议讲座": "activity", "美食餐饮": "scene", "体育运动": "activity", "园林景观": "scene", "喷泉雕塑": "object",
    "蓝天白云": "attribute", "日出日落": "attribute", "雪景冰霜": "attribute", "雨雾天气": "attribute", "逆光剪影": "attribute",
    "微距特写": "attribute", "广角视野": "attribute", "对称构图": "attribute", "倒影镜面": "attribute", "红墙黄瓦": "attribute",
    "霓虹灯光": "attribute", "烟火表演": "activity", "人群聚集": "person", "书法艺术": "object", "国旗旗帜": "object",
    "宫殿建筑": "scene", "庙宇祠堂": "scene", "塔楼钟楼": "scene", "城墙城门": "scene", "亭台楼阁": "scene",
    "码头港口": "scene", "摩天轮": "object", "高楼大厦": "scene", "别墅住宅": "scene", "农田乡村": "scene",
}

with open("/tmp/openimages_zh_translations.json", encoding="utf-8") as f:
    raw_data = json.load(f)

# 需要过滤掉的英文关键词（不适合当照片标签）
skip_keywords = [
    "organ", "syringe", "skull", "cassette deck", "fax", "punching bag",
    "tortoise", "magpie", " container", # some are too generic or weird
    "saxophone", # maybe keep? we'll be lenient
]

# 手动过滤一些翻译质量差或不适宜的标签
manual_skip = {
    "器官", "注射器", "骷髅头", "磁带机", "人头", "人类胡须", "人眼", "人毛", "指甲",
    "41.哑铃", "punching bag", "室内划船器", "天花扇", "无脊椎动物", "爬行动物", "哺乳动物",
    "传真机", "文件柜", "发雾", "punching bag", "电话机", "复印机", "室内划船器",
}

# 分类规则映射（基于原始英文中的关键词）
def classify(en, zh):
    en_l = en.lower()
    
    # scene
    if any(k in en_l for k in [
        "beach", "mountain", "park", "street", "building", "room", "kitchen", "office", "restaurant",
        "hotel", "museum", "stadium", "theater", "bridge", "tunnel", "station", "airport", "harbor",
        "desert", "forest", "lake", "river", "ocean", "farm", "garden", "playground", "zoo",
        "amusement park", "cafe", "bar", "temple", "church", "mosque", "castle", "palace",
        "bedroom", "bathroom", "living room", "dining room", "classroom", "conference room",
        "meeting room", "auditorium", "hall", "plaza", "square", "tower", "dam", "lighthouse",
        "waterfall", "canyon", "cave", "glacier", "valley", "volcano", "island", "coast",
        "port", "dock", "pier", "wharf", "marina", "yacht club", "golf course", "ski resort",
        "campground", "picnic area", "nature reserve", "national park", "city", "town", "village",
        "suburb", "countryside", "landscape", "skyline", "sunrise", "sunset", "horizon", "seascape"
    ]):
        return "scene"
    
    # food
    if any(k in en_l for k in [
        "food", "fruit", "vegetable", "meat", "bread", "cake", "dessert", "snack", "drink", "beverage",
        "coffee", "tea", "beer", "wine", "cocktail", "juice", "milk", "water", "soda", "pizza",
        "pasta", "sushi", "burger", "sandwich", "salad", "soup", "noodle", "rice", "ice cream",
        "chocolate", "cookie", "donut", "croissant", "apple", "banana", "orange", "grape",
        "strawberry", "watermelon", "mango", "pear", "peach", "lemon", "tomato", "potato",
        "carrot", "cucumber", "onion", "garlic", "mushroom", "pumpkin", "corn", "broccoli",
        "avocado", "fig", "kiwi", "coconut", "pineapple", "blueberry", "raspberry", "blackberry",
        "cheese", "yogurt", "butter", "egg", "honey", "sugar", "salt", "pepper", "spice", "herb",
        "sausage", "bacon", "ham", "steak", "chicken", "fish", "shrimp", "crab", "lobster",
        "oyster", "clam", "mussel", "scallop", "squid", "octopus", "sashimi", "dim sum",
        "dumpling", "bun", "pie", "tart", "pudding", "custard", "mousse", "gelato", "sorbet"
    ]):
        return "food"
    
    # person
    if any(k in en_l for k in [
        "person", "people", "man", "woman", "child", "baby", "girl", "boy", "infant", "toddler",
        "teenager", "adult", "senior", "elderly", "face", "human", "crowd", "audience", "group",
        "family", "couple", "friend", "parent", "mother", "father", "sibling", "brother", "sister"
    ]):
        return "person"
    
    # activity
    if any(k in en_l for k in [
        "sport", "game", "swimming", "skiing", "running", "dancing", "singing", "playing", "fishing",
        "surfing", "skateboarding", "cycling", "hiking", "camping", "picnic", "party", "wedding",
        "concert", "performance", "meeting", "lecture", "training", "exercise", "yoga", "gym",
        "basketball", "football", "soccer", "tennis", "golf", "baseball", "volleyball", "badminton",
        "boxing", "climbing", "diving", "rowing", "sailing", "racing", "competition", "tournament",
        "match", "travel", "tourism", "sightseeing", "shopping", "cooking", "eating", "drinking",
        "reading", "writing", "painting", "drawing", "photography", "filming", "recording",
        "celebration", "festival", "parade", "protest", "demonstration", "rally", "ceremony"
    ]):
        return "activity"
    
    # attribute
    if any(k in en_l for k in [
        "indoor", "outdoor", "daytime", "night", "sunset", "sunrise", "rain", "snow", "fog",
        "cloudy", "sunny", "dark", "bright", "colorful", "black and white", "vintage", "modern",
        "urban", "rural", "natural", "artificial", "clean", "dirty", "empty", "crowded", "quiet",
        "noisy", "warm", "cold", "wet", "dry", "smooth", "rough", "soft", "hard", "big", "small",
        "tall", "short", "wide", "narrow", "long", "round", "square", "symmetric", "asymmetric",
        "reflection", "shadow", "silhouette", "backlight", "neon", "fluorescent", "spotlight",
        "daylight", "twilight", "dawn", "dusk", "midnight", "noon", "morning", "afternoon",
        "evening", "summer", "winter", "spring", "autumn", "fall", "holiday", "weekend", "weekday",
        "blue sky", "white cloud", "red wall", "golden roof", "panoramic", "close-up", "macro",
        "wide-angle", "bird's eye view", "aerial view", "low angle", "high angle", "side view",
        "front view", "rear view", "profile", "three-quarter view", "dynamic", "static", "blurred",
        "sharp", "focused", "defocused", "overexposed", "underexposed", "high contrast", "low contrast",
        "vivid", "muted", "monochrome", "sepia", "vintage tone", "warm tone", "cool tone"
    ]):
        return "attribute"
    
    # 默认 object
    return "object"


# 构建分类结果
categories = {"scene": [], "object": [], "food": [], "person": [], "activity": [], "attribute": []}

# 先放入现有标签
for label, cat in existing_map.items():
    if label not in categories[cat]:
        categories[cat].append(label)

# 处理 Open Images 翻译标签
for en, zh in raw_data.items():
    zh = zh.strip().replace("\n", " ").replace("  ", " ")
    if not zh:
        continue
    if zh in manual_skip:
        continue
    if any(k.lower() in en.lower() for k in skip_keywords):
        continue
    if len(zh) > 15:
        continue
    
    cat = classify(en, zh)
    if zh not in categories[cat] and zh not in [item for sublist in categories.values() for item in sublist]:
        categories[cat].append(zh)

# 排序
for cat in categories:
    categories[cat].sort()

flat = []
for cat in ["scene", "object", "food", "person", "activity", "attribute"]:
    flat.extend(categories[cat])

stats = {k: len(v) for k, v in categories.items()}

output = {
    "metadata": {
        "version": "1.0",
        "source": "open_images_boxable + existing",
        "total": len(flat),
        "stats": stats,
    },
    "categories": categories,
    "flat": flat,
}

with open("/Users/wuzhaoqing/Pictures/photo-search-engine/config/label_space_v1.json", "w", encoding="utf-8") as f:
    json.dump(output, f, ensure_ascii=False, indent=2)

print(f"标签库生成完成：")
for k, v in stats.items():
    print(f"  {k}: {v}")
print(f"  总计: {len(flat)}")
