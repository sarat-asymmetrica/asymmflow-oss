#!/usr/bin/env python3
"""Generate professional PDFs from AsymmFlow marketing markdown docs using fpdf2."""

import os
import re
from fpdf import FPDF, XPos, YPos

DOCS_DIR = os.path.dirname(os.path.abspath(__file__))

# Use system Helvetica Neue (supports Unicode)
FONT_DIR = "/System/Library/Fonts"


def sanitize(text):
    """Replace Unicode chars that might cause issues."""
    replacements = {
        "\u2014": "--",   # em dash
        "\u2013": "-",    # en dash
        "\u2018": "'",    # left single quote
        "\u2019": "'",    # right single quote
        "\u201c": '"',    # left double quote
        "\u201d": '"',    # right double quote
        "\u2026": "...",  # ellipsis
        "\u2022": "-",    # bullet
        "\u2192": "->",   # arrow
        "\u2190": "<-",   # left arrow
        "\u25b6": ">",    # play
        "\u2502": "|",    # box drawing
        "\u251c": "|",
        "\u2514": "|",
        "\u2500": "-",
        "\u2588": "#",
        "\u2713": "[x]",  # checkmark
        "\u2717": "[ ]",  # cross
        "\u2605": "*",    # star
        "\u00b7": "-",    # middle dot
        "\u2264": "<=",
        "\u2265": ">=",
    }
    for k, v in replacements.items():
        text = text.replace(k, v)
    # Final safety - replace any remaining non-latin1
    return text.encode("latin-1", "replace").decode("latin-1")


class AsymmFlowPDF(FPDF):
    """Professional PDF generator with clean typography."""

    def __init__(self, style="traditional"):
        super().__init__()
        self.style = style
        self.set_auto_page_break(auto=True, margin=25)

    def header(self):
        if self.page_no() == 1:
            return
        self.set_font("Helvetica", "I", 8)
        self.set_text_color(150, 150, 150)
        self.cell(0, 10, sanitize("Asymmetrica AI - Confidential"), align="L")
        self.ln(4)
        self.set_draw_color(220, 220, 220)
        self.line(15, self.get_y(), 195, self.get_y())
        self.ln(6)

    def footer(self):
        self.set_y(-15)
        self.set_font("Helvetica", "I", 8)
        self.set_text_color(150, 150, 150)
        if self.page_no() > 1:
            self.cell(0, 10, str(self.page_no()), align="C")

    def add_cover_page(self, title, subtitle, meta_lines):
        self.add_page()
        self.ln(60)

        # Title
        self.set_font("Helvetica", "B", 28 if self.style == "traditional" else 32)
        self.set_text_color(10, 10, 10)
        self.multi_cell(0, 14, sanitize(title))
        self.ln(4)

        # Divider
        if self.style == "traditional":
            self.set_draw_color(180, 180, 180)
            self.set_line_width(0.5)
        else:
            self.set_draw_color(0, 0, 0)
            self.set_line_width(1)
        self.line(15, self.get_y(), 120, self.get_y())
        self.ln(8)

        # Subtitle
        self.set_font("Helvetica", "", 14)
        self.set_text_color(80, 80, 80)
        for sub_line in subtitle.split("\n"):
            self.set_x(self.l_margin)
            self.multi_cell(0, 8, sanitize(sub_line))
        self.ln(30)

        # Meta
        self.set_font("Helvetica", "", 10)
        self.set_text_color(120, 120, 120)
        for line in meta_lines:
            self.cell(0, 6, sanitize(line), new_x=XPos.LMARGIN, new_y=YPos.NEXT)

    def section_heading(self, text):
        self.ln(6)
        self.set_font("Helvetica", "B", 14)
        self.set_text_color(15, 15, 15)
        self.multi_cell(0, 8, sanitize(text))
        y = self.get_y() + 1
        if self.style == "traditional":
            self.set_draw_color(200, 200, 200)
            self.set_line_width(0.4)
        else:
            self.set_draw_color(0, 0, 0)
            self.set_line_width(0.8)
        self.line(15, y, 195, y)
        self.ln(6)

    def sub_heading(self, text):
        self.ln(3)
        self.set_font("Helvetica", "B", 11.5)
        self.set_text_color(40, 40, 40)
        self.multi_cell(0, 7, sanitize(text))
        self.ln(2)

    def sub_sub_heading(self, text):
        self.ln(2)
        self.set_font("Helvetica", "B", 10.5)
        self.set_text_color(60, 60, 60)
        self.multi_cell(0, 6, sanitize(text))
        self.ln(1)

    def body_text(self, text):
        self.set_font("Helvetica", "", 10)
        self.set_text_color(30, 30, 30)
        self.multi_cell(0, 5.5, sanitize(text))
        self.ln(2)

    def bullet_point(self, text, indent=0):
        x = 19 + indent
        self.set_font("Helvetica", "", 10)
        self.set_text_color(30, 30, 30)
        self.set_x(x)
        self.multi_cell(0, 5.5, sanitize("-  " + text))
        self.ln(1)

    def code_block(self, text):
        self.ln(2)
        if self.style == "genz":
            self.set_fill_color(30, 30, 30)
            r, g, b = 200, 200, 200
        else:
            self.set_fill_color(245, 245, 245)
            r, g, b = 50, 50, 50

        self.set_font("Courier", "", 8)
        self.set_text_color(r, g, b)

        lines = text.split("\n")
        line_h = 4.2
        block_h = len(lines) * line_h + 8

        if self.get_y() + block_h > 270:
            self.add_page()

        start_y = self.get_y()
        self.rect(15, start_y, 180, block_h, "F")

        self.set_y(start_y + 4)
        for line in lines:
            self.set_x(19)
            self.cell(0, line_h, sanitize(line), new_x=XPos.LMARGIN, new_y=YPos.NEXT)

        self.set_y(start_y + block_h + 2)
        self.ln(2)

    def add_table(self, headers, rows):
        self.ln(2)
        n_cols = len(headers)
        available_width = 180

        # Calculate widths based on content
        max_lens = []
        for i in range(n_cols):
            max_len = len(headers[i])
            for row in rows:
                if i < len(row):
                    max_len = max(max_len, len(row[i]))
            max_lens.append(max(max_len, 5))

        total = sum(max_lens)
        col_widths = [available_width * (ml / total) for ml in max_lens]
        col_widths = [max(w, 18) for w in col_widths]
        total_w = sum(col_widths)
        col_widths = [w * available_width / total_w for w in col_widths]

        est_height = (len(rows) + 1) * 8
        if self.get_y() + min(est_height, 40) > 265:
            self.add_page()

        # Header
        self.set_font("Helvetica", "B", 8)
        self.set_fill_color(25, 25, 25)
        self.set_text_color(255, 255, 255)
        for i, h in enumerate(headers):
            self.cell(col_widths[i], 7, sanitize(h), border=0, fill=True)
        self.ln()

        # Rows
        self.set_font("Helvetica", "", 8.5)
        for row_idx, row in enumerate(rows):
            if row_idx % 2 == 0:
                self.set_fill_color(249, 249, 249)
            else:
                self.set_fill_color(255, 255, 255)
            self.set_text_color(40, 40, 40)

            if self.get_y() + 7 > 270:
                self.add_page()
                self.set_font("Helvetica", "B", 8)
                self.set_fill_color(25, 25, 25)
                self.set_text_color(255, 255, 255)
                for i, h in enumerate(headers):
                    self.cell(col_widths[i], 7, sanitize(h), border=0, fill=True)
                self.ln()
                self.set_font("Helvetica", "", 8.5)
                if row_idx % 2 == 0:
                    self.set_fill_color(249, 249, 249)
                else:
                    self.set_fill_color(255, 255, 255)
                self.set_text_color(40, 40, 40)

            start_y = self.get_y()
            max_h = 0
            for i in range(n_cols):
                cell_text = row[i] if i < len(row) else ""
                x = 15 + sum(col_widths[:i])
                self.set_xy(x, start_y)
                self.multi_cell(col_widths[i], 4.5, sanitize(cell_text), border=0, fill=True)
                max_h = max(max_h, self.get_y() - start_y)

            self.set_y(start_y + max(max_h, 6))

        self.set_draw_color(200, 200, 200)
        self.line(15, self.get_y(), 195, self.get_y())
        self.ln(4)


def clean_md_formatting(text):
    """Remove markdown formatting from text."""
    text = re.sub(r"\*\*(.*?)\*\*", r"\1", text)
    text = re.sub(r"\*(.*?)\*", r"\1", text)
    text = re.sub(r"`(.*?)`", r"\1", text)
    text = re.sub(r"\[(.*?)\]\(.*?\)", r"\1", text)
    return text


def parse_markdown_to_pdf(md_path, pdf_path, style="traditional"):
    """Parse markdown and generate a styled PDF."""
    with open(md_path, "r") as f:
        lines = f.read().split("\n")

    pdf = AsymmFlowPDF(style=style)
    pdf.set_margins(15, 15, 15)

    if style == "traditional":
        pdf.add_cover_page(
            "AsymmFlow",
            "Enterprise Operations Platform\nStrategic Brief for Business Leaders",
            [
                "Prepared by: Asymmetrica AI",
                "Date: February 2026",
                "Classification: Business Confidential",
            ],
        )
    else:
        pdf.add_cover_page(
            "AsymmFlow",
            "Ship Fast. Own Your Data. Run Offline.",
            [
                "From: Asymmetrica AI",
                "Date: February 2026",
            ],
        )

    pdf.add_page()

    i = 0
    skip_until_section = True
    in_code_block = False
    code_buffer = []
    in_table = False
    table_headers = []
    table_rows = []

    while i < len(lines):
        line = lines[i]

        # Code block
        if line.strip().startswith("```"):
            if in_code_block:
                pdf.code_block("\n".join(code_buffer))
                code_buffer = []
                in_code_block = False
            else:
                if in_table:
                    pdf.add_table(table_headers, table_rows)
                    in_table = False
                    table_headers = []
                    table_rows = []
                in_code_block = True
            i += 1
            continue

        if in_code_block:
            code_buffer.append(line)
            i += 1
            continue

        # Table
        if "|" in line and line.strip().startswith("|"):
            cells = [c.strip() for c in line.strip().strip("|").split("|")]
            cells = [c for c in cells if c]
            if all(re.match(r"^[-:]+$", c) for c in cells):
                i += 1
                continue
            if not in_table:
                in_table = True
                table_headers = [clean_md_formatting(c) for c in cells]
            else:
                table_rows.append([clean_md_formatting(c) for c in cells])
            i += 1
            continue
        else:
            if in_table:
                pdf.add_table(table_headers, table_rows)
                in_table = False
                table_headers = []
                table_rows = []

        stripped = line.strip()

        # Skip front matter until first real section
        if skip_until_section:
            if stripped.startswith("## 1.") or stripped.startswith("## 1 "):
                skip_until_section = False
            else:
                i += 1
                continue

        # Headings
        if stripped.startswith("#### "):
            pdf.sub_sub_heading(stripped[5:].strip())
            i += 1
            continue
        if stripped.startswith("### "):
            pdf.sub_heading(stripped[4:].strip())
            i += 1
            continue
        if stripped.startswith("## "):
            pdf.section_heading(stripped[3:].strip())
            i += 1
            continue
        if stripped.startswith("# "):
            i += 1
            continue

        # Horizontal rule
        if stripped in ("---", "***", "___"):
            i += 1
            continue

        # Empty line
        if not stripped:
            i += 1
            continue

        # Bullet / numbered list
        if stripped.startswith("- ") or re.match(r"^\d+\.\s", stripped):
            text = re.sub(r"^[-]\s*", "", stripped)
            text = re.sub(r"^\d+\.\s*", "", text)
            text = clean_md_formatting(text)
            pdf.bullet_point(text)
            i += 1
            continue

        # Regular paragraph text
        text = clean_md_formatting(stripped)
        if text:
            para_lines = [text]
            while i + 1 < len(lines):
                next_line = lines[i + 1].strip()
                if (
                    next_line
                    and not next_line.startswith("#")
                    and not next_line.startswith("-")
                    and not next_line.startswith("|")
                    and not next_line.startswith("```")
                    and not next_line.startswith("---")
                    and not re.match(r"^\d+\.\s", next_line)
                ):
                    para_lines.append(clean_md_formatting(next_line))
                    i += 1
                else:
                    break
            pdf.body_text(" ".join(para_lines))

        i += 1

    if in_table:
        pdf.add_table(table_headers, table_rows)

    pdf.output(pdf_path)
    size_kb = os.path.getsize(pdf_path) / 1024
    print(f"  Generated: {os.path.basename(pdf_path)} ({size_kb:.0f} KB)")


if __name__ == "__main__":
    print("Generating AsymmFlow marketing PDFs...")
    print()
    parse_markdown_to_pdf(
        os.path.join(DOCS_DIR, "AsymmFlow_Enterprise_Brief_Traditional.md"),
        os.path.join(DOCS_DIR, "AsymmFlow_Enterprise_Brief_Traditional.pdf"),
        style="traditional",
    )
    parse_markdown_to_pdf(
        os.path.join(DOCS_DIR, "AsymmFlow_Enterprise_Brief_GenZ.md"),
        os.path.join(DOCS_DIR, "AsymmFlow_Enterprise_Brief_GenZ.pdf"),
        style="genz",
    )
    print()
    print("Done. Both PDFs ready in docs/")
