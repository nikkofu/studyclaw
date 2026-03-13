class RecitationLineAnalysis {
  const RecitationLineAnalysis({
    required this.index,
    required this.expected,
    required this.observed,
    required this.matchRatio,
    required this.status,
    required this.notes,
  });

  factory RecitationLineAnalysis.fromJson(Map<String, dynamic> json) {
    return RecitationLineAnalysis(
      index: _readInt(json['index']),
      expected: json['expected']?.toString() ?? '',
      observed: json['observed']?.toString() ?? '',
      matchRatio: _readDouble(json['match_ratio']),
      status: json['status']?.toString() ?? 'partial',
      notes: json['notes']?.toString() ?? '',
    );
  }

  final int index;
  final String expected;
  final String observed;
  final double matchRatio;
  final String status;
  final String notes;

  bool get isMatched => status == 'matched';
  bool get isPartial => status == 'partial';
  bool get isMissing => status == 'missing';

  String get statusLabel {
    switch (status) {
      case 'matched':
        return '基本对上';
      case 'missing':
        return '差距较大';
      case 'partial':
      default:
        return '部分对上';
    }
  }

  String get ratioLabel => '${(matchRatio * 100).round()}%';
}

class RecitationAnalysis {
  const RecitationAnalysis({
    required this.status,
    required this.parserMode,
    required this.scene,
    required this.recognizedTitle,
    required this.recognizedAuthor,
    required this.referenceTitle,
    required this.referenceAuthor,
    required this.referenceText,
    required this.normalizedTranscript,
    required this.reconstructedText,
    required this.completionRatio,
    required this.needsRetry,
    required this.summary,
    required this.suggestion,
    required this.issues,
    required this.matchedLines,
  });

  factory RecitationAnalysis.fromJson(Map<String, dynamic> json) {
    return RecitationAnalysis(
      status: json['status']?.toString() ?? 'success',
      parserMode: json['parser_mode']?.toString() ?? 'rule_fallback',
      scene: json['scene']?.toString() ?? 'recitation',
      recognizedTitle: json['recognized_title']?.toString() ?? '',
      recognizedAuthor: json['recognized_author']?.toString() ?? '',
      referenceTitle: json['reference_title']?.toString() ?? '',
      referenceAuthor: json['reference_author']?.toString() ?? '',
      referenceText: json['reference_text']?.toString() ?? '',
      normalizedTranscript: json['normalized_transcript']?.toString() ?? '',
      reconstructedText: json['reconstructed_text']?.toString() ?? '',
      completionRatio: _readDouble(json['completion_ratio']),
      needsRetry: json['needs_retry'] == true,
      summary: json['summary']?.toString() ?? '',
      suggestion: json['suggestion']?.toString() ?? '',
      issues: ((json['issues'] as List<dynamic>? ?? const <dynamic>[]))
          .map((item) => item.toString())
          .where((item) => item.trim().isNotEmpty)
          .toList(),
      matchedLines:
          ((json['matched_lines'] as List<dynamic>? ?? const <dynamic>[]))
              .map((item) =>
                  RecitationLineAnalysis.fromJson(item as Map<String, dynamic>))
              .toList(),
    );
  }

  final String status;
  final String parserMode;
  final String scene;
  final String recognizedTitle;
  final String recognizedAuthor;
  final String referenceTitle;
  final String referenceAuthor;
  final String referenceText;
  final String normalizedTranscript;
  final String reconstructedText;
  final double completionRatio;
  final bool needsRetry;
  final String summary;
  final String suggestion;
  final List<String> issues;
  final List<RecitationLineAnalysis> matchedLines;

  String get displayTitle {
    if (recognizedTitle.trim().isNotEmpty) {
      return recognizedTitle.trim();
    }
    if (referenceTitle.trim().isNotEmpty) {
      return referenceTitle.trim();
    }
    return '未识别标题';
  }

  String get displayAuthor {
    if (recognizedAuthor.trim().isNotEmpty) {
      return recognizedAuthor.trim();
    }
    if (referenceAuthor.trim().isNotEmpty) {
      return referenceAuthor.trim();
    }
    return '作者待识别';
  }

  String get completionLabel => '${(completionRatio * 100).round()}%';

  String get parserModeLabel =>
      parserMode == 'llm_hybrid' ? 'LLM 混合分析' : '规则兜底分析';
}

double _readDouble(Object? value) {
  if (value is double) {
    return value;
  }
  if (value is int) {
    return value.toDouble();
  }
  if (value is num) {
    return value.toDouble();
  }
  return double.tryParse(value?.toString() ?? '') ?? 0;
}

int _readInt(Object? value) {
  if (value is int) {
    return value;
  }
  if (value is num) {
    return value.toInt();
  }
  return int.tryParse(value?.toString() ?? '') ?? 0;
}
