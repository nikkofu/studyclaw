enum WordPlaybackLanguage { english, chinese }

extension WordPlaybackLanguageX on WordPlaybackLanguage {
  String get label {
    switch (this) {
      case WordPlaybackLanguage.english:
        return '英语';
      case WordPlaybackLanguage.chinese:
        return '语文';
    }
  }

  String get localeCode {
    switch (this) {
      case WordPlaybackLanguage.english:
        return 'en-US';
      case WordPlaybackLanguage.chinese:
        return 'zh-CN';
    }
  }

  String get hintText {
    switch (this) {
      case WordPlaybackLanguage.english:
        return '每行一个单词，例如：apple';
      case WordPlaybackLanguage.chinese:
        return '每行一个词语，例如：认真';
    }
  }

  List<String> get sampleWords {
    switch (this) {
      case WordPlaybackLanguage.english:
        return const <String>['apple', 'library', 'notebook', 'tomorrow'];
      case WordPlaybackLanguage.chinese:
        return const <String>['认真', '练习', '进步', '勇敢'];
    }
  }

  static WordPlaybackLanguage fromString(String value) {
    if (value.toLowerCase().contains('chinese') || value.contains('语文')) {
      return WordPlaybackLanguage.chinese;
    }
    return WordPlaybackLanguage.english;
  }
}

class WordItem {
  const WordItem({
    required this.index,
    required this.text,
    this.meaning,
    this.hint,
  });

  factory WordItem.fromJson(Map<String, dynamic> json) {
    return WordItem(
      index: _readInt(json['index']),
      text: json['text']?.toString() ?? '',
      meaning: json['meaning']?.toString(),
      hint: json['hint']?.toString(),
    );
  }

  final int index;
  final String text;
  final String? meaning;
  final String? hint;
}

class WordList {
  const WordList({
    required this.wordListId,
    required this.familyId,
    required this.childId,
    required this.assignedDate,
    required this.title,
    required this.language,
    required this.items,
    required this.totalItems,
  });

  factory WordList.fromJson(Map<String, dynamic> json) {
    return WordList(
      wordListId: json['word_list_id']?.toString() ?? '',
      familyId: _readInt(json['family_id']),
      childId: _readInt(json['child_id']),
      assignedDate: json['assigned_date']?.toString() ?? '',
      title: json['title']?.toString() ?? '',
      language: WordPlaybackLanguageX.fromString(json['language']?.toString() ?? 'english'),
      items: ((json['items'] as List<dynamic>? ?? const <dynamic>[]))
          .map((item) => WordItem.fromJson(item as Map<String, dynamic>))
          .toList(),
      totalItems: _readInt(json['total_items']),
    );
  }

  final String wordListId;
  final int familyId;
  final int childId;
  final String assignedDate;
  final String title;
  final WordPlaybackLanguage language;
  final List<WordItem> items;
  final int totalItems;
}

class DictationSession {
  const DictationSession({
    required this.sessionId,
    required this.wordListId,
    required this.status,
    required this.currentIndex,
    required this.totalItems,
    required this.playedCount,
    required this.completedItems,
    this.currentItem,
  });

  factory DictationSession.fromJson(Map<String, dynamic> json) {
    return DictationSession(
      sessionId: json['session_id']?.toString() ?? '',
      wordListId: json['word_list_id']?.toString() ?? '',
      status: json['status']?.toString() ?? 'active',
      currentIndex: _readInt(json['current_index']),
      totalItems: _readInt(json['total_items']),
      playedCount: _readInt(json['played_count']),
      completedItems: _readInt(json['completed_items']),
      currentItem: json['current_item'] != null
          ? WordItem.fromJson(json['current_item'] as Map<String, dynamic>)
          : null,
    );
  }

  final String sessionId;
  final String wordListId;
  final String status;
  final int currentIndex;
  final int totalItems;
  final int playedCount;
  final int completedItems;
  final WordItem? currentItem;

  bool get isCompleted => status == 'completed';
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

String buildWordSampleText(WordPlaybackLanguage language) {
  return language.sampleWords.join('\n');
}

List<String> parseWordEntries(String rawText) {
  return rawText
      .split(RegExp(r'[\n,，;；]'))
      .map((item) => item.trim())
      .where((item) => item.isNotEmpty)
      .toList();
}
