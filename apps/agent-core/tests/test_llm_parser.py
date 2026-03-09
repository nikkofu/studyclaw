import unittest

from services.llm_parser import extract_structure_outline, parse_parent_input_fallback


SAMPLE_GROUP_MESSAGE = """数学3.6：
1、校本P14～15
2、练习册P12～13

英：
1. 背默M1U1知识梳理单小作文
2. 部分学生继续订正1号本
3. 预习M1U2
（1）书本上标注好“黄页”出现单词的音标
（2）抄写单词（今天默写全对，可免抄）
（3）沪学习听录音跟读

语文：
1. 背作文
2. 练习卷
"""


class LLMParserFallbackTests(unittest.TestCase):
    def test_extract_structure_outline_detects_sections_and_signals(self):
        outline = extract_structure_outline(SAMPLE_GROUP_MESSAGE)

        self.assertEqual(outline["detected_subjects"], ["数学", "英语", "语文"])
        self.assertIn("subject_headings", outline["format_signals"])
        self.assertIn("numbered_tasks", outline["format_signals"])
        self.assertIn("nested_subtasks", outline["format_signals"])
        self.assertEqual(len(outline["tasks"]), 9)

    def test_parse_parent_input_fallback_merges_nested_subtasks(self):
        result = parse_parent_input_fallback(SAMPLE_GROUP_MESSAGE)
        expected_tasks = [
            ("数学", "校本P14～15", "校本P14～15"),
            ("数学", "练习册P12～13", "练习册P12～13"),
            ("英语", "背默M1U1知识梳理单小作文", "背默M1U1知识梳理单小作文"),
            ("英语", "部分学生继续订正1号本", "部分学生继续订正1号本"),
            ("英语", "预习M1U2", "书本上标注好“黄页”出现单词的音标"),
            ("英语", "预习M1U2", "抄写单词（今天默写全对，可免抄）"),
            ("英语", "预习M1U2", "沪学习听录音跟读"),
            ("语文", "背作文", "背作文"),
            ("语文", "练习卷", "练习卷"),
        ]

        self.assertEqual(result["status"], "success")
        self.assertEqual(result["parser_mode"], "rule_fallback")
        self.assertEqual(len(result["data"]), 9)
        self.assertGreater(result["analysis"]["needs_review_count"], 0)
        self.assertEqual(
            [(item["subject"], item["group_title"], item["title"]) for item in result["data"]],
            expected_tasks,
        )
        self.assertTrue(result["data"][3]["needs_review"])
        self.assertTrue(result["data"][5]["needs_review"])


if __name__ == "__main__":
    unittest.main()
