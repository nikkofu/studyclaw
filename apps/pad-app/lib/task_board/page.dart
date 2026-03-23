import 'dart:convert';
import 'dart:async';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:pad_app/ui_kit/kid_theme.dart';
import 'package:pad_app/ui_kit/kid_card.dart';
import 'package:pad_app/ui_kit/kid_components.dart';
import 'package:pad_app/task_board/controller.dart';
import 'package:pad_app/task_board/daily_stats.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/recitation_analysis.dart';
import 'package:pad_app/task_board/repository.dart';
import 'package:pad_app/voice_commands/models.dart';
import 'package:pad_app/voice_commands/speech_recognizer.dart';
import 'package:pad_app/voice_commands/speech_recognizer_contract.dart';
import 'package:pad_app/word_playback/controller.dart';
import 'package:pad_app/word_playback/models.dart';
import 'package:pad_app/word_playback/speaker.dart';

const String defaultApiBaseUrl = String.fromEnvironment('API_BASE_URL',
    defaultValue: 'http://localhost:8080');

enum _PadHomeTab { tasks, words }

enum _JourneyStepVisualState { complete, active, pending, failed }

enum _SpeechWorkbenchMode { command, transcript, companion }

enum _LearningScene {
  taskFinish,
  dictation,
  recitation,
  reading,
  conversation,
  journal,
  companion,
}

const Object _missingVoiceValue = Object();

class _JourneyStepData {
  const _JourneyStepData({
    required this.title,
    required this.caption,
    required this.state,
    this.timeLabel,
  });

  final String title;
  final String caption;
  final _JourneyStepVisualState state;
  final String? timeLabel;
}

String _formatJourneyTime(String? raw) {
  final value = raw?.trim() ?? '';
  if (value.isEmpty) return '等待中';

  final parsed = DateTime.tryParse(value);
  if (parsed == null) return value;
  final local = parsed.toLocal();
  final hour = local.hour.toString().padLeft(2, '0');
  final minute = local.minute.toString().padLeft(2, '0');
  return '$hour:$minute';
}

String _formatSubmissionClock(DateTime submittedAt) {
  final local = submittedAt.toLocal();
  final hour = local.hour.toString().padLeft(2, '0');
  final minute = local.minute.toString().padLeft(2, '0');
  return '提交 $hour:$minute';
}

String _formatSubmissionBytes(int byteCount) {
  if (byteCount < 1024) {
    return '大小 $byteCount B';
  }

  final kilobytes = byteCount / 1024;
  if (kilobytes < 1024) {
    final value = kilobytes >= 10
        ? kilobytes.toStringAsFixed(0)
        : kilobytes.toStringAsFixed(1);
    return '大小 $value KB';
  }

  final megabytes = kilobytes / 1024;
  return '大小 ${megabytes.toStringAsFixed(1)} MB';
}

String _shortTraceToken(String? raw) {
  final value = raw?.trim() ?? '';
  if (value.isEmpty) return '';
  return value.length <= 8 ? value : value.substring(0, 8);
}

extension on _SpeechWorkbenchMode {
  String get label {
    switch (this) {
      case _SpeechWorkbenchMode.command:
        return '快捷指令';
      case _SpeechWorkbenchMode.transcript:
        return '长段记录';
      case _SpeechWorkbenchMode.companion:
        return '陪伴监听';
    }
  }

  String get description {
    switch (this) {
      case _SpeechWorkbenchMode.command:
        return '适合“好了”“下一个”“数学订正好了”这类短句，结束后直接执行动作。';
      case _SpeechWorkbenchMode.transcript:
        return '适合背诵、朗读、对话、口述日记，系统会持续收听并自动断句。';
      case _SpeechWorkbenchMode.companion:
        return '适合较长时间陪伴学习，像会议记录一样实时沉淀孩子整段表达。';
    }
  }

  Color get accentColor {
    switch (this) {
      case _SpeechWorkbenchMode.command:
        return KidColors.color2;
      case _SpeechWorkbenchMode.transcript:
        return KidColors.color4;
      case _SpeechWorkbenchMode.companion:
        return KidColors.color3;
    }
  }

  IconData get icon {
    switch (this) {
      case _SpeechWorkbenchMode.command:
        return Icons.flash_on_rounded;
      case _SpeechWorkbenchMode.transcript:
        return Icons.notes_rounded;
      case _SpeechWorkbenchMode.companion:
        return Icons.favorite_rounded;
    }
  }
}

extension on _LearningScene {
  String get label {
    switch (this) {
      case _LearningScene.taskFinish:
        return '任务完成';
      case _LearningScene.dictation:
        return '单词默写';
      case _LearningScene.recitation:
        return '背诵';
      case _LearningScene.reading:
        return '朗读';
      case _LearningScene.conversation:
        return '对话';
      case _LearningScene.journal:
        return '口述日记';
      case _LearningScene.companion:
        return '学习陪伴';
    }
  }

  String get description {
    switch (this) {
      case _LearningScene.taskFinish:
        return '做完一项任务就说出来，让系统帮你完成确认。';
      case _LearningScene.dictation:
        return '默写完成一个词时，直接说“好了”“下一个”“继续”。';
      case _LearningScene.recitation:
        return '适合背课文、古诗、定义、公式。';
      case _LearningScene.reading:
        return '适合英语朗读、语文朗读和整段跟读。';
      case _LearningScene.conversation:
        return '适合口语问答、角色扮演和自由表达。';
      case _LearningScene.journal:
        return '适合口述日记、复盘今天学到了什么。';
      case _LearningScene.companion:
        return '适合较长陪伴过程，边学边记，不急着结束。';
    }
  }

  List<String> get sampleUtterances {
    switch (this) {
      case _LearningScene.taskFinish:
        return const <String>[
          '数学订正好了',
          '一课一练做完了',
          '全部都好了',
        ];
      case _LearningScene.dictation:
        return const <String>[
          '好了',
          '下一个',
          '继续',
          'Next',
        ];
      case _LearningScene.recitation:
        return const <String>[
          '我先背第一段',
          '这一句我再来一次',
          '请继续记录',
        ];
      case _LearningScene.reading:
        return const <String>[
          'The weather is sunny today.',
          '我开始朗读第一段',
          '这句再读一遍',
        ];
      case _LearningScene.conversation:
        return const <String>[
          '今天学校里发生了什么？',
          '我觉得这道题难在这里',
          '我们继续下一题',
        ];
      case _LearningScene.journal:
        return const <String>[
          '今天我最开心的是...',
          '我觉得最难的是...',
          '我准备明天继续努力',
        ];
      case _LearningScene.companion:
        return const <String>[
          '我现在开始做默写',
          '这道题我改好了',
          '我们继续下一部分',
        ];
    }
  }
}

class _SpeechSegment {
  const _SpeechSegment({
    required this.index,
    required this.text,
    required this.capturedAt,
    this.isLive = false,
  });

  final int index;
  final String text;
  final DateTime capturedAt;
  final bool isLive;
}

String _normalizeVoiceTranscript(String raw) {
  return raw.replaceAll(RegExp(r'\s+'), ' ').trim();
}

bool _containsCjk(String value) {
  return RegExp(r'[\u3400-\u9FFF]').hasMatch(value);
}

bool _isComparableSpeechChar(String value) {
  return RegExp(r'[\u3400-\u9FFFA-Za-z0-9]').hasMatch(value);
}

String _compactComparableSpeechText(String value) {
  final buffer = StringBuffer();
  for (var index = 0; index < value.length; index += 1) {
    final char = value[index];
    if (_isComparableSpeechChar(char)) {
      buffer.write(char);
    }
  }
  return buffer.toString();
}

List<int> _buildComparableSpeechIndexMap(String value) {
  final indices = <int>[];
  for (var index = 0; index < value.length; index += 1) {
    final char = value[index];
    if (_isComparableSpeechChar(char)) {
      indices.add(index + 1);
    }
  }
  return indices;
}

List<String> _extractReferenceGuidanceUnits(
  String referenceText, {
  required _LearningScene scene,
}) {
  final normalizedLines = referenceText
      .replaceAll('\r\n', '\n')
      .split('\n')
      .map(_normalizeVoiceTranscript)
      .where((item) => item.isNotEmpty)
      .toList(growable: false);
  if (normalizedLines.isEmpty) {
    return const <String>[];
  }

  if (scene != _LearningScene.reading) {
    return normalizedLines;
  }

  final sentenceUnits = <String>[];
  const breakChars = '。！？!?；;';
  for (final line in normalizedLines) {
    final buffer = StringBuffer();
    for (var index = 0; index < line.length; index += 1) {
      final char = line[index];
      buffer.write(char);
      if (breakChars.contains(char)) {
        final segment = _normalizeVoiceTranscript(buffer.toString());
        if (segment.isNotEmpty) {
          sentenceUnits.add(segment);
        }
        buffer.clear();
      }
    }
    final tail = _normalizeVoiceTranscript(buffer.toString());
    if (tail.isNotEmpty) {
      sentenceUnits.add(tail);
    }
  }

  return sentenceUnits.isNotEmpty ? sentenceUnits : normalizedLines;
}

List<String> _splitSpeechChunksByReferenceShape(
  String transcript, {
  required _LearningScene scene,
  required String referenceText,
}) {
  final normalizedTranscript = _normalizeVoiceTranscript(transcript);
  if (normalizedTranscript.isEmpty ||
      !_containsCjk(normalizedTranscript) ||
      RegExp(r'[。！？!?；;]').hasMatch(normalizedTranscript)) {
    return const <String>[];
  }

  final referenceUnits = _extractReferenceGuidanceUnits(
    referenceText,
    scene: scene,
  )
      .map(_compactComparableSpeechText)
      .where((item) => item.isNotEmpty)
      .toList(growable: false);
  if (referenceUnits.length < 2) {
    return const <String>[];
  }

  final compactTranscript = _compactComparableSpeechText(normalizedTranscript);
  final indexMap = _buildComparableSpeechIndexMap(normalizedTranscript);
  if (compactTranscript.isEmpty ||
      compactTranscript.length != indexMap.length ||
      compactTranscript.length < referenceUnits.length * 2) {
    return const <String>[];
  }

  final totalExpectedLength = referenceUnits.fold<int>(
    0,
    (sum, item) => sum + item.length,
  );
  if (totalExpectedLength == 0) {
    return const <String>[];
  }

  final segments = <String>[];
  var compactStart = 0;
  var originalStart = 0;
  var consumedExpectedLength = 0;

  for (var unitIndex = 0; unitIndex < referenceUnits.length; unitIndex += 1) {
    final unitLength = referenceUnits[unitIndex].length;
    final isLastUnit = unitIndex == referenceUnits.length - 1;
    final compactEnd = isLastUnit
        ? compactTranscript.length
        : () {
            final remainingUnits = referenceUnits.length - unitIndex - 1;
            final remainingComparableChars =
                compactTranscript.length - compactStart;
            final remainingExpectedLength =
                totalExpectedLength - consumedExpectedLength;
            final suggestedLength = (remainingComparableChars *
                    unitLength /
                    remainingExpectedLength)
                .round();
            final maxCurrentLength = remainingComparableChars - remainingUnits;
            final boundedLength = suggestedLength.clamp(1, maxCurrentLength);
            return compactStart + boundedLength;
          }();

    if (compactEnd <= compactStart || compactEnd > indexMap.length) {
      return const <String>[];
    }

    final originalEnd = indexMap[compactEnd - 1];
    final segment = _normalizeVoiceTranscript(
      normalizedTranscript.substring(originalStart, originalEnd),
    );
    if (segment.isEmpty) {
      return const <String>[];
    }

    segments.add(segment);
    compactStart = compactEnd;
    originalStart = originalEnd;
    consumedExpectedLength += unitLength;
  }

  return segments;
}

List<String> _splitSpeechChunks(
  String transcript, {
  required _SpeechWorkbenchMode mode,
  required _LearningScene scene,
  String? referenceText,
}) {
  final normalized = _normalizeVoiceTranscript(transcript);
  if (normalized.isEmpty) {
    return const <String>[];
  }

  if (mode == _SpeechWorkbenchMode.command) {
    return <String>[normalized];
  }

  final rawReferenceText = (referenceText ?? '').trim();
  if (rawReferenceText.isNotEmpty) {
    final referenceGuided = _splitSpeechChunksByReferenceShape(
      normalized,
      scene: scene,
      referenceText: rawReferenceText,
    );
    if (referenceGuided.isNotEmpty) {
      return referenceGuided;
    }
  }

  final punctuationSegments = <String>[];
  final buffer = StringBuffer();
  const breakChars = '。！？!?；;，,\n';

  for (var index = 0; index < normalized.length; index += 1) {
    final char = normalized[index];
    buffer.write(char);
    if (breakChars.contains(char)) {
      final segment = _normalizeVoiceTranscript(buffer.toString());
      if (segment.isNotEmpty) {
        punctuationSegments.add(segment);
      }
      buffer.clear();
    }
  }

  final tail = _normalizeVoiceTranscript(buffer.toString());
  if (tail.isNotEmpty) {
    punctuationSegments.add(tail);
  }

  final sourceSegments = punctuationSegments.isNotEmpty
      ? punctuationSegments
      : <String>[normalized];
  final result = <String>[];
  for (final segment in sourceSegments) {
    result.addAll(_chunkLongSpeechSegment(segment, scene: scene));
  }
  return result.where((item) => item.trim().isNotEmpty).toList();
}

List<String> _chunkLongSpeechSegment(
  String segment, {
  required _LearningScene scene,
}) {
  final normalized = _normalizeVoiceTranscript(segment);
  if (normalized.isEmpty) {
    return const <String>[];
  }

  final words = normalized.split(' ').where((item) => item.trim().isNotEmpty);
  if (words.length >= 16) {
    final result = <String>[];
    final buffer = <String>[];
    for (final word in words) {
      buffer.add(word);
      if (buffer.length >= 12) {
        result.add(buffer.join(' '));
        buffer.clear();
      }
    }
    if (buffer.isNotEmpty) {
      result.add(buffer.join(' '));
    }
    return result;
  }

  if (_containsCjk(normalized)) {
    switch (scene) {
      case _LearningScene.dictation:
      case _LearningScene.taskFinish:
        return <String>[normalized];
      case _LearningScene.recitation:
      case _LearningScene.reading:
      case _LearningScene.conversation:
      case _LearningScene.journal:
      case _LearningScene.companion:
        // Prefer keeping CJK speech in one piece unless punctuation or
        // recognizer pause boundaries already split it for us.
        return <String>[normalized];
    }
  }

  return <String>[normalized];
}

String _extractLiveSegmentText(
  String previewTranscript,
  List<_SpeechSegment> committedSegments,
) {
  final preview = _normalizeVoiceTranscript(previewTranscript);
  if (preview.isEmpty) {
    return '';
  }

  final committed = committedSegments
      .map((item) => _normalizeVoiceTranscript(item.text))
      .where((item) => item.isNotEmpty)
      .join(' ')
      .trim();
  if (committed.isEmpty) {
    return preview;
  }
  if (preview == committed || committed.endsWith(preview)) {
    return '';
  }
  if (preview.startsWith(committed)) {
    return preview.substring(committed.length).trim();
  }
  return preview;
}

List<_SpeechSegment> _buildSpeechSegmentsFromRecognizer(
  List<_SpeechSegment> committedSegments, {
  required String previewTranscript,
  required bool isListening,
  required DateTime fallbackTime,
  required _SpeechWorkbenchMode mode,
  required _LearningScene scene,
  String? referenceText,
}) {
  if (!isListening) {
    final sourceList =
        committedSegments.where((item) => !item.isLive).toList(growable: false);
    if (sourceList.isEmpty) {
      final normalizedPreview = _normalizeVoiceTranscript(previewTranscript);
      final refinedChunks = _splitSpeechChunks(
        normalizedPreview,
        mode: mode,
        scene: scene,
        referenceText: referenceText,
      );
      if (refinedChunks.length > 1) {
        return List<_SpeechSegment>.generate(refinedChunks.length, (index) {
          return _SpeechSegment(
            index: index + 1,
            text: refinedChunks[index],
            capturedAt: fallbackTime.add(Duration(seconds: index * 8)),
            isLive: false,
          );
        });
      }
    }
  }

  final result = <_SpeechSegment>[];
  for (final item in committedSegments) {
    final chunks = _splitSpeechChunks(
      item.text,
      mode: mode,
      scene: scene,
      referenceText: null,
    );
    if (chunks.isEmpty) {
      continue;
    }
    for (final chunk in chunks) {
      result.add(
        _SpeechSegment(
          index: result.length + 1,
          text: chunk,
          capturedAt: item.capturedAt,
          isLive: false,
        ),
      );
    }
  }

  final liveText =
      _extractLiveSegmentText(previewTranscript, committedSegments);
  if (liveText.isNotEmpty && isListening) {
    result.add(
      _SpeechSegment(
        index: result.length + 1,
        text: liveText,
        capturedAt: fallbackTime,
        isLive: true,
      ),
    );
  }

  if (result.isEmpty && previewTranscript.trim().isNotEmpty) {
    return _buildSpeechSegments(
      previewTranscript,
      mode: mode,
      scene: scene,
      baseTime: fallbackTime,
      isListening: isListening,
      referenceText: referenceText,
    );
  }
  return result;
}

List<_SpeechSegment> _buildSpeechSegments(
  String transcript, {
  required _SpeechWorkbenchMode mode,
  required _LearningScene scene,
  required DateTime baseTime,
  required bool isListening,
  String? referenceText,
}) {
  final chunks = _splitSpeechChunks(
    transcript,
    mode: mode,
    scene: scene,
    referenceText: !isListening ? referenceText : null,
  );
  return List<_SpeechSegment>.generate(chunks.length, (index) {
    return _SpeechSegment(
      index: index + 1,
      text: chunks[index],
      capturedAt: baseTime.add(Duration(seconds: index * 8)),
      isLive: isListening && index == chunks.length - 1,
    );
  });
}

String _formatVoiceClock(Duration duration) {
  final minutes = duration.inMinutes.toString().padLeft(2, '0');
  final seconds = (duration.inSeconds % 60).toString().padLeft(2, '0');
  return '$minutes:$seconds';
}

String _formatVoiceSegmentTime(DateTime value) {
  final local = value.toLocal();
  final hour = local.hour.toString().padLeft(2, '0');
  final minute = local.minute.toString().padLeft(2, '0');
  final second = local.second.toString().padLeft(2, '0');
  return '$hour:$minute:$second';
}

int _countVoiceCharacters(String transcript) {
  return transcript.replaceAll(RegExp(r'\s+'), '').length;
}

String _buildVoiceSummary({
  required _SpeechWorkbenchMode mode,
  required _LearningScene scene,
  required List<_SpeechSegment> segments,
  required String transcript,
}) {
  final charCount = _countVoiceCharacters(transcript);
  switch (mode) {
    case _SpeechWorkbenchMode.command:
      return '本次识别到 $charCount 个字，会在结束说话后统一理解并执行对应动作。';
    case _SpeechWorkbenchMode.transcript:
      if (scene == _LearningScene.recitation ||
          scene == _LearningScene.reading) {
        return '已按真实停顿记录成 ${segments.length} 段，共 $charCount 个字；上方保留孩子真实开口节奏，下方再继续做标准原文逐句对照。';
      }
      return '已按真实停顿记录成 ${segments.length} 段，共 $charCount 个字，后续可继续做复述检查、朗读纠音或学习复盘。';
    case _SpeechWorkbenchMode.companion:
      return '陪伴记录已按真实停顿整理成 ${segments.length} 段，共 $charCount 个字，适合继续追踪孩子整段学习过程并提取关键问题。';
  }
}

String _buildVoiceEncouragement({
  required _SpeechWorkbenchMode mode,
  required _LearningScene scene,
}) {
  switch (scene) {
    case _LearningScene.taskFinish:
      return '做完就主动告诉系统，这是在认真管理自己的学习节奏，做得很好。';
    case _LearningScene.dictation:
      return mode == _SpeechWorkbenchMode.command
          ? '你说得很清楚，下一词也能稳稳拿下。'
          : '一词一词坚持下来，本身就是很棒的专注力训练。';
    case _LearningScene.recitation:
      return '敢把内容完整说出来，就是在同时训练记忆、表达和自信。';
    case _LearningScene.reading:
      return '愿意一段一段读出来，进步会比闷着更快更扎实。';
    case _LearningScene.conversation:
      return '愿意主动开口对话，就是语言能力真正长出来的开始。';
    case _LearningScene.journal:
      return '能把自己的想法讲清楚，是非常宝贵的整理能力。';
    case _LearningScene.companion:
      return '你不是一个人在坚持，这段努力都会被好好记录下来。';
  }
}

List<_JourneyStepData> _buildJourneySteps(DictationSession session) {
  final stageKey = resolveDictationWorkerStage(session);

  _JourneyStepVisualState receiveState;
  switch (stageKey) {
    case 'queued':
      receiveState = _JourneyStepVisualState.active;
      break;
    case 'mark_processing_failed':
      receiveState = _JourneyStepVisualState.failed;
      break;
    case 'idle':
      receiveState = _JourneyStepVisualState.pending;
      break;
    default:
      receiveState = _JourneyStepVisualState.complete;
  }

  _JourneyStepVisualState compareState;
  switch (stageKey) {
    case 'processing':
    case 'loading_word_list':
    case 'llm_grading':
      compareState = _JourneyStepVisualState.active;
      break;
    case 'load_word_list_failed':
    case 'llm_grading_failed':
      compareState = _JourneyStepVisualState.failed;
      break;
    case 'completed':
    case 'persist_result_failed':
      compareState = _JourneyStepVisualState.complete;
      break;
    default:
      compareState = receiveState == _JourneyStepVisualState.complete
          ? _JourneyStepVisualState.pending
          : _JourneyStepVisualState.pending;
  }

  _JourneyStepVisualState resultState;
  switch (stageKey) {
    case 'completed':
      resultState = _JourneyStepVisualState.complete;
      break;
    case 'persist_result_failed':
    case 'failed':
      resultState = _JourneyStepVisualState.failed;
      break;
    default:
      resultState = _JourneyStepVisualState.pending;
  }

  return <_JourneyStepData>[
    _JourneyStepData(
      title: '拍照完成',
      caption: '照片已经准备好，正在送去批改。',
      state: _JourneyStepVisualState.complete,
      timeLabel: _formatJourneyTime(
          session.gradingRequestedAt ?? session.updatedAt ?? session.startedAt),
    ),
    _JourneyStepData(
      title: '云端接收',
      caption: '后台接到这次交卷请求。',
      state: receiveState,
      timeLabel: _formatJourneyTime(session.gradingRequestedAt),
    ),
    _JourneyStepData(
      title: 'AI 比对',
      caption: 'AI 看照片、核对正确答案。',
      state: compareState,
      timeLabel: compareState == _JourneyStepVisualState.complete ||
              compareState == _JourneyStepVisualState.failed ||
              compareState == _JourneyStepVisualState.active
          ? _formatJourneyTime(session.gradingCompletedAt ??
              session.updatedAt ??
              session.gradingRequestedAt)
          : null,
    ),
    _JourneyStepData(
      title: '结果回到平板',
      caption: '最终结果同步回当前设备。',
      state: resultState,
      timeLabel: resultState == _JourneyStepVisualState.complete ||
              resultState == _JourneyStepVisualState.failed
          ? _formatJourneyTime(session.gradingCompletedAt ?? session.updatedAt)
          : null,
    ),
  ];
}

List<String> _buildTracePills(DictationSession session) {
  final pills = <String>[
    if (session.sessionId.trim().isNotEmpty) '会话 ${session.sessionId}',
    if (session.gradingResult?.gradingId.trim().isNotEmpty == true)
      '批改 ${session.gradingResult!.gradingId}',
    if (_shortTraceToken(session.debugContext?.photoSha1).isNotEmpty)
      '图片 ${_shortTraceToken(session.debugContext?.photoSha1)}',
  ];
  return pills;
}

class _VoiceAssistantState {
  const _VoiceAssistantState({
    this.mode = _SpeechWorkbenchMode.command,
    this.scene = _LearningScene.taskFinish,
    this.isListening = false,
    this.isResolving = false,
    this.lastTranscript,
    this.summaryMessage,
    this.encouragementMessage,
    this.noticeMessage,
    this.errorMessage,
    this.lastResolution,
    this.recitationAnalysis,
    this.liveSegmentText,
    this.sessionStartedAt,
    this.sessionFinishedAt,
    this.segments = const <_SpeechSegment>[],
  });

  final _SpeechWorkbenchMode mode;
  final _LearningScene scene;
  final bool isListening;
  final bool isResolving;
  final String? lastTranscript;
  final String? summaryMessage;
  final String? encouragementMessage;
  final String? noticeMessage;
  final String? errorMessage;
  final VoiceCommandResolution? lastResolution;
  final RecitationAnalysis? recitationAnalysis;
  final String? liveSegmentText;
  final DateTime? sessionStartedAt;
  final DateTime? sessionFinishedAt;
  final List<_SpeechSegment> segments;

  bool get isBusy => isListening || isResolving;

  bool get hasTranscript => (lastTranscript?.trim().isNotEmpty ?? false);

  _VoiceAssistantState copyWith({
    _SpeechWorkbenchMode? mode,
    _LearningScene? scene,
    bool? isListening,
    bool? isResolving,
    Object? lastTranscript = _missingVoiceValue,
    Object? summaryMessage = _missingVoiceValue,
    Object? encouragementMessage = _missingVoiceValue,
    Object? noticeMessage = _missingVoiceValue,
    Object? errorMessage = _missingVoiceValue,
    Object? lastResolution = _missingVoiceValue,
    Object? recitationAnalysis = _missingVoiceValue,
    Object? liveSegmentText = _missingVoiceValue,
    Object? sessionStartedAt = _missingVoiceValue,
    Object? sessionFinishedAt = _missingVoiceValue,
    Object? segments = _missingVoiceValue,
  }) {
    return _VoiceAssistantState(
      mode: mode ?? this.mode,
      scene: scene ?? this.scene,
      isListening: isListening ?? this.isListening,
      isResolving: isResolving ?? this.isResolving,
      lastTranscript: lastTranscript == _missingVoiceValue
          ? this.lastTranscript
          : lastTranscript as String?,
      summaryMessage: summaryMessage == _missingVoiceValue
          ? this.summaryMessage
          : summaryMessage as String?,
      encouragementMessage: encouragementMessage == _missingVoiceValue
          ? this.encouragementMessage
          : encouragementMessage as String?,
      noticeMessage: noticeMessage == _missingVoiceValue
          ? this.noticeMessage
          : noticeMessage as String?,
      errorMessage: errorMessage == _missingVoiceValue
          ? this.errorMessage
          : errorMessage as String?,
      lastResolution: lastResolution == _missingVoiceValue
          ? this.lastResolution
          : lastResolution as VoiceCommandResolution?,
      recitationAnalysis: recitationAnalysis == _missingVoiceValue
          ? this.recitationAnalysis
          : recitationAnalysis as RecitationAnalysis?,
      liveSegmentText: liveSegmentText == _missingVoiceValue
          ? this.liveSegmentText
          : liveSegmentText as String?,
      sessionStartedAt: sessionStartedAt == _missingVoiceValue
          ? this.sessionStartedAt
          : sessionStartedAt as DateTime?,
      sessionFinishedAt: sessionFinishedAt == _missingVoiceValue
          ? this.sessionFinishedAt
          : sessionFinishedAt as DateTime?,
      segments: segments == _missingVoiceValue
          ? this.segments
          : segments as List<_SpeechSegment>,
    );
  }
}

class PadTaskBoardPage extends StatefulWidget {
  const PadTaskBoardPage(
      {super.key,
      this.autoLoad = true,
      this.initialDate,
      this.initialApiBaseUrl,
      this.initialFamilyId,
      this.initialUserId,
      this.repository = const RemoteTaskBoardRepository(),
      this.wordPlaybackController,
      this.speechRecognizer});
  final bool autoLoad;
  final String? initialDate, initialApiBaseUrl;
  final int? initialFamilyId, initialUserId;
  final TaskBoardRepository repository;
  final WordPlaybackController? wordPlaybackController;
  final SpeechRecognizer? speechRecognizer;
  @override
  State<PadTaskBoardPage> createState() => _PadTaskBoardPageState();
}

class _PadTaskBoardPageState extends State<PadTaskBoardPage>
    with SingleTickerProviderStateMixin {
  late final AnimationController _celebrationController;
  int _selectedSubjectIndex = 0;
  late final TextEditingController _apiBaseUrlController,
      _familyIdController,
      _userIdController,
      _dateController,
      _wordListController,
      _voiceReferenceController;
  late final TaskBoardController _controller;
  late final WordPlaybackController _wordController;
  late final bool _ownsWordController;
  late final SpeechRecognizer _speechRecognizer;
  late final bool _ownsSpeechRecognizer;
  _PadHomeTab _selectedTab = _PadHomeTab.tasks;
  _VoiceAssistantState _voiceAssistantState = const _VoiceAssistantState();
  TaskBoardRequest? _activeVoiceRequest;
  VoiceCommandSurface? _activeVoiceSurface;
  Timer? _voiceSessionTicker;
  bool _encouragementVoiceEnabled = true;
  String? _lastTaskEncouragementVoiceKey;
  String? _lastVoiceWorkbenchEncouragementKey;

  @override
  void initState() {
    super.initState();
    _celebrationController =
        AnimationController(vsync: this, duration: const Duration(seconds: 1));
    _apiBaseUrlController = TextEditingController(
        text: widget.initialApiBaseUrl ?? defaultApiBaseUrl);
    _familyIdController =
        TextEditingController(text: '${widget.initialFamilyId ?? 306}');
    _userIdController =
        TextEditingController(text: '${widget.initialUserId ?? 1}');
    _dateController = TextEditingController(
        text: widget.initialDate ?? formatTaskBoardDate(DateTime.now()));
    _wordListController = TextEditingController();
    _voiceReferenceController = TextEditingController();
    _controller = TaskBoardController(repository: widget.repository);
    _controller.addListener(_handleTaskBoardStateChanged);
    if (widget.wordPlaybackController != null) {
      _wordController = widget.wordPlaybackController!;
      _ownsWordController = false;
    } else {
      _wordController = WordPlaybackController(
          speaker: createWordSpeaker(), repository: widget.repository);
      _ownsWordController = true;
    }
    if (widget.speechRecognizer != null) {
      _speechRecognizer = widget.speechRecognizer!;
      _ownsSpeechRecognizer = false;
    } else {
      _speechRecognizer = createSpeechRecognizer();
      _ownsSpeechRecognizer = true;
    }
    if (widget.autoLoad) {
      scheduleMicrotask(() => _loadBoard(showLoadingState: true));
    }
  }

  @override
  void dispose() {
    _celebrationController.dispose();
    _apiBaseUrlController.dispose();
    _familyIdController.dispose();
    _userIdController.dispose();
    _dateController.dispose();
    _wordListController.dispose();
    _voiceReferenceController.dispose();
    _controller.removeListener(_handleTaskBoardStateChanged);
    _controller.dispose();
    if (_ownsWordController) {
      _wordController.dispose();
    }
    if (_ownsSpeechRecognizer) {
      unawaited(_speechRecognizer.stop());
    }
    _voiceSessionTicker?.cancel();
    super.dispose();
  }

  TaskBoardRequest? _buildRequest() {
    final req = TaskBoardRequest(
        apiBaseUrl: _apiBaseUrlController.text.trim(),
        familyId: int.tryParse(_familyIdController.text.trim()) ?? 0,
        userId: int.tryParse(_userIdController.text.trim()) ?? 0,
        date: _dateController.text.trim());
    return req.validate() == null ? req : null;
  }

  void _loadBoard({bool showLoadingState = false}) {
    final r = _buildRequest();
    if (r != null) {
      _controller.loadBoard(r, showLoadingState: showLoadingState);
    }
  }

  Future<void> _refreshBoard() async {
    final r = _buildRequest();
    if (r != null) {
      await _controller.refresh(r);
    }
  }

  Future<void> _startRecommendedTask() async {
    final board = _controller.state.board;
    if (board == null) {
      return;
    }
    final target = _controller.resolveLaunchTask(board);
    if (target == null) {
      return;
    }
    if (target.completed) {
      return;
    }
    await _updateSingleTask(target, true);
  }

  void _handleTaskBoardStateChanged() {
    if (!mounted) {
      return;
    }

    final state = _controller.state;
    final message = _taskEncouragementMessage(state)?.trim() ?? '';
    if (message.isEmpty) {
      _lastTaskEncouragementVoiceKey = null;
      return;
    }
    if (message == _lastTaskEncouragementVoiceKey) {
      return;
    }

    _lastTaskEncouragementVoiceKey = message;
    if (!_encouragementVoiceEnabled || !_wordController.supportsPlayback) {
      return;
    }
    if (state.noticeTone == TaskBoardNoticeTone.info) {
      return;
    }

    unawaited(_wordController.speakCoachMessage(message));
  }

  void _toggleEncouragementVoice() {
    if (!_wordController.supportsPlayback) {
      return;
    }
    setState(() {
      _encouragementVoiceEnabled = !_encouragementVoiceEnabled;
    });
  }

  Future<void> _replayTaskEncouragement() async {
    final message = _taskEncouragementMessage(_controller.state)?.trim() ?? '';
    if (message.isEmpty) {
      return;
    }
    await _wordController.speakCoachMessage(message);
  }

  Future<void> _replayVoiceWorkbenchEncouragement() async {
    final message = _voiceAssistantState.encouragementMessage?.trim() ?? '';
    if (message.isEmpty) {
      return;
    }
    await _wordController.speakCoachMessage(message);
  }

  void _maybeAutoSpeakVoiceWorkbenchEncouragement(String? message) {
    final normalized = message?.trim() ?? '';
    if (normalized.isEmpty) {
      _lastVoiceWorkbenchEncouragementKey = null;
      return;
    }

    final key = '${_voiceAssistantState.scene.name}|'
        '${_voiceAssistantState.sessionFinishedAt?.millisecondsSinceEpoch ?? 0}|'
        '$normalized';
    if (key == _lastVoiceWorkbenchEncouragementKey) {
      return;
    }
    _lastVoiceWorkbenchEncouragementKey = key;

    if (!_encouragementVoiceEnabled || !_wordController.supportsPlayback) {
      return;
    }

    unawaited(_wordController.speakCoachMessage(normalized));
  }

  void _setSelectedTab(_PadHomeTab tab) {
    final nextMode = _voiceAssistantState.mode;
    setState(() {
      _selectedTab = tab;
      _voiceAssistantState = _voiceAssistantState.copyWith(
        scene: _normalizedVoiceScene(
          nextMode,
          _voiceAssistantState.scene,
          tab,
        ),
      );
    });
  }

  Future<void> _updateSingleTask(TaskItem task, bool completed) async {
    final r = _buildRequest();
    if (r == null) {
      return;
    }
    await _controller.updateSingleTask(r, task, completed);
    if (completed &&
        _controller.state.board?.tasks.every((t) => t.completed) == true) {
      _celebrationController.forward(from: 0);
    }
  }

  Future<void> _updateAllTasks(bool completed) async {
    final r = _buildRequest();
    if (r == null) {
      return;
    }
    await _controller.updateAllTasks(r, completed: completed);
    if (completed) {
      _celebrationController.forward(from: 0);
    }
  }

  Future<void> _updateSubjectGroup(TaskGroup group, bool completed) async {
    final r = _buildRequest();
    if (r == null) {
      return;
    }
    await _controller.updateSubjectGroup(r, group, completed);
  }

  Future<void> _updateHomeworkGroup(
    HomeworkGroup group,
    bool completed,
  ) async {
    final r = _buildRequest();
    if (r == null) {
      return;
    }
    await _controller.updateHomeworkGroup(r, group, completed);
  }

  Future<void> _takeAndSubmitPhoto() async {
    final picker = ImagePicker();
    final photo = await picker.pickImage(
        source: ImageSource.camera, maxWidth: 1024, imageQuality: 85);
    if (photo == null) return;
    final r = _buildRequest();
    if (r == null) return;

    // Auto-start session if not active
    if (_wordController.state.session == null) {
      await _wordController.startDictation(r);
    }

    final submittedAt = DateTime.now();
    final bytes = await photo.readAsBytes();
    final base64Image = base64Encode(bytes);
    await _wordController.submitPhotoForGrading(
      r.apiBaseUrl,
      base64Image,
      previewBytes: bytes,
      submittedAt: submittedAt,
    );
  }

  void _openSettingsSheet() {
    showModalBottomSheet<void>(
        context: context,
        isScrollControlled: true,
        builder: (ctx) => KidBottomSheetFrame(
            title: '同步配置',
            child: Column(children: [
              TextField(
                  controller: _apiBaseUrlController,
                  decoration: const InputDecoration(labelText: 'API 地址')),
              const SizedBox(height: 16),
              Row(children: [
                Expanded(
                    child: TextField(
                        controller: _familyIdController,
                        decoration: const InputDecoration(labelText: '家庭 ID'))),
                const SizedBox(width: 16),
                Expanded(
                    child: TextField(
                        controller: _userIdController,
                        decoration: const InputDecoration(labelText: '孩子 ID')))
              ]),
              const SizedBox(height: 24),
              KidActionBtn(
                  label: '立即同步',
                  color: KidColors.color1,
                  onTap: () {
                    Navigator.pop(ctx);
                    _loadBoard(showLoadingState: true);
                  }),
            ])));
  }

  VoiceCommandSurface get _currentVoiceSurface {
    return _selectedTab == _PadHomeTab.words
        ? VoiceCommandSurface.dictation
        : VoiceCommandSurface.taskBoard;
  }

  String _voiceLocaleForSurface(VoiceCommandSurface surface) {
    if (surface == VoiceCommandSurface.dictation) {
      return _wordController.state.language.localeCode;
    }
    return 'zh-CN';
  }

  List<_LearningScene> _availableVoiceScenes(
    _SpeechWorkbenchMode mode,
    _PadHomeTab tab,
  ) {
    switch (mode) {
      case _SpeechWorkbenchMode.command:
        return <_LearningScene>[
          tab == _PadHomeTab.words
              ? _LearningScene.dictation
              : _LearningScene.taskFinish,
        ];
      case _SpeechWorkbenchMode.transcript:
        return const <_LearningScene>[
          _LearningScene.recitation,
          _LearningScene.reading,
          _LearningScene.conversation,
          _LearningScene.journal,
        ];
      case _SpeechWorkbenchMode.companion:
        return const <_LearningScene>[
          _LearningScene.companion,
          _LearningScene.dictation,
          _LearningScene.reading,
          _LearningScene.recitation,
        ];
    }
  }

  _LearningScene _defaultVoiceSceneForMode(
    _SpeechWorkbenchMode mode,
    _PadHomeTab tab,
  ) {
    return _availableVoiceScenes(mode, tab).first;
  }

  _LearningScene _normalizedVoiceScene(
    _SpeechWorkbenchMode mode,
    _LearningScene scene,
    _PadHomeTab tab,
  ) {
    final availableScenes = _availableVoiceScenes(mode, tab);
    if (availableScenes.contains(scene)) {
      return scene;
    }
    return availableScenes.first;
  }

  String _voiceLocaleForWorkbench({
    required _SpeechWorkbenchMode mode,
    required _LearningScene scene,
  }) {
    if (mode == _SpeechWorkbenchMode.command) {
      return _voiceLocaleForSurface(_currentVoiceSurface);
    }
    if (scene == _LearningScene.dictation) {
      return _wordController.state.language.localeCode;
    }
    if (scene == _LearningScene.reading &&
        _selectedTab == _PadHomeTab.words &&
        _wordController.state.language == WordPlaybackLanguage.english) {
      return _wordController.state.language.localeCode;
    }
    return 'zh-CN';
  }

  bool _supportsReferenceAnalysis(_VoiceAssistantState state) {
    if (state.mode == _SpeechWorkbenchMode.command) {
      return false;
    }
    return state.scene == _LearningScene.recitation ||
        state.scene == _LearningScene.reading;
  }

  String? _preferredVoiceReferenceSubject() {
    final board = _controller.state.board;
    if (_selectedTab != _PadHomeTab.tasks ||
        board == null ||
        board.groups.isEmpty) {
      return null;
    }
    final index =
        _selectedSubjectIndex < board.groups.length ? _selectedSubjectIndex : 0;
    return board.groups[index].subject;
  }

  bool _taskTypeMatchesScene(TaskItem task, _LearningScene scene) {
    final type = task.taskType.trim().toLowerCase();
    switch (scene) {
      case _LearningScene.recitation:
        return type == 'recitation' ||
            type == 'memorization' ||
            type == 'memorize' ||
            type == 'poem_recitation' ||
            type == 'classical_poem';
      case _LearningScene.reading:
        return type == 'reading' ||
            type == 'read_aloud' ||
            type == '朗读' ||
            type == 'follow_reading';
      default:
        return false;
    }
  }

  bool _taskSupportsLearningScene(TaskItem task, _LearningScene scene) {
    if (!task.hasReferenceMaterial) {
      return false;
    }
    if (_taskTypeMatchesScene(task, scene)) {
      return true;
    }

    final haystack = <String>[
      task.subject,
      task.groupTitle,
      task.content,
      task.referenceTitle,
      task.analysisMode,
    ].join(' ').toLowerCase();

    switch (scene) {
      case _LearningScene.recitation:
        return haystack.contains('背诵') ||
            haystack.contains('背默') ||
            haystack.contains('古诗') ||
            haystack.contains('诗词') ||
            haystack.contains('课文') ||
            haystack.contains('朗诵');
      case _LearningScene.reading:
        return haystack.contains('朗读') ||
            haystack.contains('跟读') ||
            haystack.contains('阅读') ||
            haystack.contains('read aloud');
      default:
        return false;
    }
  }

  int _referenceTaskScore(
    TaskItem task,
    _LearningScene scene, {
    String? preferredSubject,
  }) {
    var score = 0;
    if (!task.completed) {
      score += 4;
    }
    if (preferredSubject != null && task.subject == preferredSubject) {
      score += 5;
    }
    if (_taskTypeMatchesScene(task, scene)) {
      score += 6;
    }
    if (task.analysisMode.trim().isNotEmpty) {
      score += 1;
    }
    if (task.referenceTitle.trim().isNotEmpty) {
      score += 1;
    }
    return score;
  }

  TaskItem? _currentReferenceTaskForState(_VoiceAssistantState state) {
    if (!_supportsReferenceAnalysis(state)) {
      return null;
    }

    final board = _controller.state.board;
    if (board == null || board.tasks.isEmpty) {
      return null;
    }

    final preferredSubject = _preferredVoiceReferenceSubject();
    final candidates = board.tasks
        .where((task) => _taskSupportsLearningScene(task, state.scene))
        .toList(growable: false);
    if (candidates.isEmpty) {
      return null;
    }

    final ranked = candidates.toList()
      ..sort((left, right) {
        final rightScore = _referenceTaskScore(
          right,
          state.scene,
          preferredSubject: preferredSubject,
        );
        final leftScore = _referenceTaskScore(
          left,
          state.scene,
          preferredSubject: preferredSubject,
        );
        if (rightScore != leftScore) {
          return rightScore.compareTo(leftScore);
        }
        return left.taskId.compareTo(right.taskId);
      });
    return ranked.first;
  }

  bool _showManualReferenceInput(_VoiceAssistantState state) {
    if (!_supportsReferenceAnalysis(state)) {
      return false;
    }
    return _currentReferenceTaskForState(state) == null;
  }

  String _effectiveReferenceText(_VoiceAssistantState state) {
    final task = _currentReferenceTaskForState(state);
    if (task != null && task.referenceText.trim().isNotEmpty) {
      return task.referenceText.trim();
    }
    return _voiceReferenceController.text.trim();
  }

  String? _taskReferenceSummaryTitle(_VoiceAssistantState state) {
    final task = _currentReferenceTaskForState(state);
    if (task == null) {
      return null;
    }
    final title = task.referenceTitle.trim().isNotEmpty
        ? task.referenceTitle.trim()
        : task.content.trim();
    return state.scene == _LearningScene.reading
        ? '当前朗读任务：$title'
        : '当前背诵任务：$title';
  }

  String? _taskReferenceSummaryDetail(_VoiceAssistantState state) {
    final task = _currentReferenceTaskForState(state);
    if (task == null) {
      return null;
    }
    final author = task.referenceAuthor.trim();
    final authorLine = author.isEmpty ? '' : '作者：$author。';
    if (task.hideReferenceFromChild) {
      return '系统已从任务板自动带入隐藏参考内容。$authorLine 孩子端不展开正文，结束说话后会自动做标题识别和逐句比对。';
    }
    return '系统已从任务板自动带入参考内容。$authorLine 结束说话后会直接对照这条任务的标准文本。';
  }

  String _referenceInputLabel(_VoiceAssistantState state) {
    return state.scene == _LearningScene.reading ? '家长/老师朗读原文' : '家长/老师背诵原文';
  }

  String _referenceInputHint(_VoiceAssistantState state) {
    if (state.scene == _LearningScene.reading) {
      return '这块更适合由家长/老师预先录入。结束说话时会对照朗读内容，找出不稳的句子。';
    }
    return '这块更适合在布置任务时由家长/老师预置。结束说话时会自动识别标题并逐句比对，孩子端不必长期暴露原文。';
  }

  bool _shouldRunRecitationAnalysis(
      _SpeechWorkbenchMode mode, _LearningScene scene) {
    if (mode == _SpeechWorkbenchMode.command) {
      return false;
    }
    return scene == _LearningScene.recitation ||
        scene == _LearningScene.reading;
  }

  Future<void> _persistVoiceLearningSession({
    required _SpeechWorkbenchMode mode,
    required _LearningScene scene,
    required List<_SpeechSegment> segments,
    required String transcript,
    required String summaryMessage,
    required String encouragementMessage,
    required DateTime startedAt,
    required DateTime finishedAt,
    RecitationAnalysis? recitationAnalysis,
  }) async {
    if (!_hasUsableApiBaseUrl()) {
      return;
    }
    final request = _buildRequest();
    if (request == null) {
      return;
    }
    final referenceTask = _currentReferenceTaskForState(_voiceAssistantState);
    await widget.repository.saveVoiceLearningSession(
      _apiBaseUrlController.text.trim(),
      payload: {
        'family_id': request.familyId,
        'child_id': request.userId,
        'assigned_date': request.date,
        'mode': mode.name,
        'scene': scene.name,
        if (referenceTask != null) 'task_id': referenceTask.taskId,
        if (referenceTask != null) 'task_title': referenceTask.content,
        if (referenceTask != null) 'task_type': referenceTask.taskType,
        if (referenceTask != null) 'reference_title': referenceTask.referenceTitle,
        if (referenceTask != null) 'reference_author': referenceTask.referenceAuthor,
        if (referenceTask != null) 'reference_source': referenceTask.referenceSource,
        if (referenceTask != null)
          'hide_reference_from_child': referenceTask.hideReferenceFromChild,
        'merged_transcript': transcript,
        'summary': summaryMessage,
        'encouragement': encouragementMessage,
        'started_at': startedAt.toUtc().toIso8601String(),
        'ended_at': finishedAt.toUtc().toIso8601String(),
        'transcript_segments': segments
            .where((item) => item.text.trim().isNotEmpty)
            .map((item) => {
                  'sequence': item.index,
                  'started_at': item.capturedAt.toUtc().toIso8601String(),
                  'ended_at': item.capturedAt.toUtc().toIso8601String(),
                  'transcript': item.text,
                  'source': item.isLive ? 'live' : 'recognizer',
                })
            .toList(),
        if (recitationAnalysis != null)
          'analysis': {
            'recognized_title': recitationAnalysis.recognizedTitle,
            'recognized_author': recitationAnalysis.recognizedAuthor,
            'reference_title': recitationAnalysis.referenceTitle,
            'reference_author': recitationAnalysis.referenceAuthor,
            'completion_ratio': recitationAnalysis.completionRatio,
            'needs_retry': recitationAnalysis.needsRetry,
            'summary': recitationAnalysis.summary,
            'suggestion': recitationAnalysis.suggestion,
            'issues': recitationAnalysis.issues,
            'parser_mode': recitationAnalysis.parserMode,
            'normalized_transcript': recitationAnalysis.normalizedTranscript,
            'matched_lines': recitationAnalysis.matchedLines
                .map((line) => {
                      'index': line.index,
                      'expected': line.expected,
                      'observed': line.observed,
                      'match_ratio': line.matchRatio,
                      'status': line.status,
                      'notes': line.notes,
                    })
                .toList(),
          },
      },
    );
  }

  Map<String, String> _buildRecitationAnalysisMetadata(
      {TaskItem? referenceTask}) {
    return <String, String>{
      'surface': _selectedTab.name,
      'workbench_mode': _voiceAssistantState.mode.name,
      'scene': _voiceAssistantState.scene.name,
      'reference_source': referenceTask == null ? 'manual' : 'task',
      if (referenceTask != null) 'task_id': '${referenceTask.taskId}',
      if (referenceTask != null &&
          referenceTask.referenceSource.trim().isNotEmpty)
        'reference_task_source': referenceTask.referenceSource.trim(),
      if (referenceTask != null && referenceTask.taskType.trim().isNotEmpty)
        'task_type': referenceTask.taskType.trim(),
      if (referenceTask != null && referenceTask.analysisMode.trim().isNotEmpty)
        'analysis_mode': referenceTask.analysisMode.trim(),
    };
  }

  bool _hasUsableApiBaseUrl() {
    final parsed = Uri.tryParse(_apiBaseUrlController.text.trim());
    return parsed != null &&
        (parsed.scheme == 'http' || parsed.scheme == 'https') &&
        parsed.host.isNotEmpty;
  }

  List<String> _voiceSamplesForState(_VoiceAssistantState state) {
    if (state.mode == _SpeechWorkbenchMode.command) {
      return _currentVoiceSurface.sampleUtterances;
    }
    return state.scene.sampleUtterances;
  }

  String _voiceHintForState(_VoiceAssistantState state) {
    if (!_speechRecognizer.supportsRecognition) {
      return '当前设备暂不支持语音识别。';
    }

    if (state.isListening) {
      switch (state.mode) {
        case _SpeechWorkbenchMode.command:
          return '已经进入持续收听。中间停几秒不会结束，想执行动作时再点“结束说话”。';
        case _SpeechWorkbenchMode.transcript:
          return '正在持续记录整段表达。系统会边听边断句，结束时再整理成完整学习记录。';
        case _SpeechWorkbenchMode.companion:
          return '陪伴监听进行中。只要你不主动结束，它就会继续像学习记录员一样陪着你。';
      }
    }

    switch (state.mode) {
      case _SpeechWorkbenchMode.command:
        return '适合短句控制。点击开始后可持续收听，结束时统一理解并执行。';
      case _SpeechWorkbenchMode.transcript:
        return '适合背诵、朗读、对话、口述日记，结束时会自动整理出分段记录。';
      case _SpeechWorkbenchMode.companion:
        return '适合更长时间的学习陪伴和实时笔记，边学边说也能持续记录。';
    }
  }

  Duration _voiceSessionDuration(_VoiceAssistantState state) {
    final startedAt = state.sessionStartedAt;
    if (startedAt == null) {
      return Duration.zero;
    }
    final endAt = state.isListening
        ? DateTime.now()
        : state.sessionFinishedAt ?? DateTime.now();
    return endAt.difference(startedAt);
  }

  void _startVoiceSessionTicker() {
    _voiceSessionTicker?.cancel();
    _voiceSessionTicker = Timer.periodic(const Duration(seconds: 1), (_) {
      if (!mounted || !_voiceAssistantState.isListening) {
        return;
      }
      setState(() {});
    });
  }

  void _stopVoiceSessionTicker() {
    _voiceSessionTicker?.cancel();
    _voiceSessionTicker = null;
  }

  void _setVoiceWorkbenchMode(_SpeechWorkbenchMode mode) {
    if (_voiceAssistantState.isBusy) {
      return;
    }
    final scene = _defaultVoiceSceneForMode(mode, _selectedTab);
    setState(() {
      _voiceAssistantState = _VoiceAssistantState(
        mode: mode,
        scene: scene,
        noticeMessage: '已切换到${mode.label}模式。',
      );
    });
  }

  void _setVoiceLearningScene(_LearningScene scene) {
    if (_voiceAssistantState.isBusy) {
      return;
    }
    setState(() {
      _voiceAssistantState = _VoiceAssistantState(
        mode: _voiceAssistantState.mode,
        scene: scene,
        noticeMessage: '已切换到${scene.label}场景。',
      );
    });
  }

  void _clearVoiceWorkbench() {
    if (_voiceAssistantState.isBusy) {
      return;
    }
    final mode = _voiceAssistantState.mode;
    final scene =
        _normalizedVoiceScene(mode, _voiceAssistantState.scene, _selectedTab);
    setState(() {
      _voiceAssistantState = _VoiceAssistantState(
        mode: mode,
        scene: scene,
        noticeMessage: '已清空这次语音记录，可以重新开始。',
      );
    });
  }

  Future<void> _toggleVoiceAssistant() async {
    if (_voiceAssistantState.isListening) {
      await _finishVoiceAssistant();
      return;
    }

    await _startVoiceAssistant();
  }

  Future<void> _startVoiceAssistant() async {
    final mode = _voiceAssistantState.mode;
    final scene =
        _normalizedVoiceScene(mode, _voiceAssistantState.scene, _selectedTab);
    TaskBoardRequest? request;
    VoiceCommandSurface? surface;

    if (mode == _SpeechWorkbenchMode.command) {
      request = _buildRequest();
      if (request == null) {
        setState(() {
          _voiceAssistantState = _voiceAssistantState.copyWith(
            errorMessage: '请先确认 API、家庭 ID、孩子 ID 和日期配置。',
            noticeMessage: null,
          );
        });
        return;
      }

      surface = _currentVoiceSurface;
      if (surface == VoiceCommandSurface.taskBoard &&
          _controller.state.board == null) {
        setState(() {
          _voiceAssistantState = _voiceAssistantState.copyWith(
            errorMessage: '请先同步任务板，再使用快捷指令。',
            noticeMessage: null,
          );
        });
        return;
      }
      if (surface == VoiceCommandSurface.dictation &&
          !_wordController.state.hasWords) {
        setState(() {
          _voiceAssistantState = _voiceAssistantState.copyWith(
            errorMessage: '请先同步词单或开启听写，再使用默写快捷指令。',
            noticeMessage: null,
          );
        });
        return;
      }
    }

    final locale = _voiceLocaleForWorkbench(mode: mode, scene: scene);
    final startedAt = DateTime.now();
    _activeVoiceRequest = request;
    _activeVoiceSurface = surface;
    setState(() {
      _voiceAssistantState = _voiceAssistantState.copyWith(
        mode: mode,
        scene: scene,
        isListening: true,
        isResolving: false,
        lastTranscript: null,
        summaryMessage: null,
        encouragementMessage: null,
        recitationAnalysis: null,
        liveSegmentText: null,
        sessionStartedAt: startedAt,
        sessionFinishedAt: null,
        segments: const <_SpeechSegment>[],
        lastResolution: null,
        errorMessage: null,
        noticeMessage: mode == _SpeechWorkbenchMode.command
            ? '已开始持续收听，想执行动作时再点“结束说话”。'
            : '已开始持续记录，想收尾时再点“结束说话”。',
      );
    });
    _startVoiceSessionTicker();

    try {
      await _speechRecognizer.startListening(
        locale: locale,
        onTranscriptChanged: (transcript) {
          if (!mounted) {
            return;
          }
          final normalized = _normalizeVoiceTranscript(transcript);
          final startedAt =
              _voiceAssistantState.sessionStartedAt ?? DateTime.now();
          final committedSegments = _voiceAssistantState.segments
              .where((item) => !item.isLive)
              .toList(growable: false);
          final referenceText = _effectiveReferenceText(_voiceAssistantState);
          final segments = _buildSpeechSegmentsFromRecognizer(
            committedSegments,
            previewTranscript: normalized,
            mode: _voiceAssistantState.mode,
            scene: _voiceAssistantState.scene,
            isListening: true,
            fallbackTime: startedAt,
            referenceText: referenceText,
          );
          setState(() {
            _voiceAssistantState = _voiceAssistantState.copyWith(
              lastTranscript: normalized,
              segments: segments,
              liveSegmentText: _extractLiveSegmentText(
                normalized,
                committedSegments,
              ),
              errorMessage: null,
              noticeMessage:
                  _voiceAssistantState.mode == _SpeechWorkbenchMode.command
                      ? '正在持续收听，点“结束说话”后再统一理解动作。'
                      : '正在持续记录，系统会边听边整理断句。',
            );
          });
        },
        onSegmentCommitted: (segment) {
          if (!mounted) {
            return;
          }
          final normalizedSegment = _normalizeVoiceTranscript(segment);
          if (normalizedSegment.isEmpty) {
            return;
          }
          setState(() {
            final committedSegments = _voiceAssistantState.segments
                .where((item) => !item.isLive)
                .toList(growable: true);
            committedSegments.add(
              _SpeechSegment(
                index: committedSegments.length + 1,
                text: normalizedSegment,
                capturedAt: DateTime.now(),
              ),
            );
            _voiceAssistantState = _voiceAssistantState.copyWith(
              segments: committedSegments,
              liveSegmentText: null,
              noticeMessage:
                  _voiceAssistantState.mode == _SpeechWorkbenchMode.command
                      ? '已捕获一个自然停顿片段，继续说完后再结束。'
                      : '已按真实停顿记录一段内容，继续说时会接着记。',
            );
          });
        },
      );
    } catch (error) {
      _stopVoiceSessionTicker();
      _activeVoiceRequest = null;
      _activeVoiceSurface = null;
      if (!mounted) return;
      setState(() {
        _voiceAssistantState = _voiceAssistantState.copyWith(
          isListening: false,
          isResolving: false,
          errorMessage: '语音指令失败：$error',
          noticeMessage: null,
          liveSegmentText: null,
          sessionFinishedAt: DateTime.now(),
        );
      });
    }
  }

  Future<void> _finishVoiceAssistant() async {
    final mode = _voiceAssistantState.mode;
    final scene = _voiceAssistantState.scene;
    final request = _activeVoiceRequest;
    final surface = _activeVoiceSurface;
    if (mode == _SpeechWorkbenchMode.command &&
        (request == null || surface == null)) {
      await _speechRecognizer.stop();
      _stopVoiceSessionTicker();
      if (!mounted) {
        return;
      }
      setState(() {
        _voiceAssistantState = _voiceAssistantState.copyWith(
          isListening: false,
          isResolving: false,
          noticeMessage: '语音识别已取消。',
          errorMessage: null,
          liveSegmentText: null,
          sessionFinishedAt: DateTime.now(),
        );
      });
      return;
    }

    _stopVoiceSessionTicker();
    if (mounted) {
      setState(() {
        _voiceAssistantState = _voiceAssistantState.copyWith(
          isListening: false,
          isResolving: mode == _SpeechWorkbenchMode.command,
          errorMessage: null,
          noticeMessage: mode == _SpeechWorkbenchMode.command
              ? '正在整理整段语音内容...'
              : '正在整理这次学习语音记录...',
        );
      });
    }

    try {
      final transcript = await _speechRecognizer.finishListening();
      final normalized = _normalizeVoiceTranscript(transcript.transcript);
      final finishedAt = DateTime.now();
      final startedAt = _voiceAssistantState.sessionStartedAt ?? finishedAt;
      var segments = _buildSpeechSegmentsFromRecognizer(
        _voiceAssistantState.segments.where((item) => !item.isLive).toList(),
        previewTranscript: normalized,
        mode: mode,
        scene: scene,
        isListening: false,
        fallbackTime: finishedAt,
        referenceText: _effectiveReferenceText(_voiceAssistantState),
      );
      if (segments.isEmpty && normalized.isNotEmpty) {
        segments = _buildSpeechSegments(
          normalized,
          mode: mode,
          scene: scene,
          baseTime: startedAt,
          isListening: false,
          referenceText: _effectiveReferenceText(_voiceAssistantState),
        );
      }
      if (!mounted) {
        return;
      }

      if (mode != _SpeechWorkbenchMode.command) {
        final summaryMessage = _buildVoiceSummary(
          mode: mode,
          scene: scene,
          segments: segments,
          transcript: normalized,
        );
        final encouragementMessage = _buildVoiceEncouragement(
          mode: mode,
          scene: scene,
        );
        var noticeMessage = '本次${scene.label}记录已经整理好了。';
        RecitationAnalysis? recitationAnalysis;

        if (_shouldRunRecitationAnalysis(mode, scene)) {
          final referenceTask =
              _currentReferenceTaskForState(_voiceAssistantState);
          final referenceText = _effectiveReferenceText(_voiceAssistantState);
          if (_hasUsableApiBaseUrl()) {
            setState(() {
              _voiceAssistantState = _voiceAssistantState.copyWith(
                isListening: false,
                isResolving: true,
                lastTranscript: normalized,
                segments: segments,
                summaryMessage: summaryMessage,
                encouragementMessage: encouragementMessage,
                sessionFinishedAt: finishedAt,
                errorMessage: null,
                noticeMessage: scene == _LearningScene.recitation
                    ? '正在识别诗题并对照原文...'
                    : '正在对照朗读原文...',
                lastResolution: null,
                recitationAnalysis: null,
                liveSegmentText: null,
              );
            });

            try {
              recitationAnalysis = await widget.repository.analyzeRecitation(
                _apiBaseUrlController.text.trim(),
                transcript: normalized,
                scene: scene.name,
                locale: _voiceLocaleForWorkbench(mode: mode, scene: scene),
                referenceText: referenceText,
                metadata: _buildRecitationAnalysisMetadata(
                    referenceTask: referenceTask),
              );
              noticeMessage = recitationAnalysis.needsRetry
                  ? '背诵比对完成，建议按标记句子再熟读一遍后重背。'
                  : '背诵比对完成，这一遍整体比较稳。';
            } catch (error) {
              noticeMessage = '本地记录已经保留，但云端背诵比对暂时失败：$error';
            }
          } else {
            noticeMessage = '本地记录已经整理好，但当前 API 地址无效，暂时无法做背诵对照。';
          }
        }

        setState(() {
          _voiceAssistantState = _voiceAssistantState.copyWith(
            isListening: false,
            isResolving: false,
            lastTranscript: normalized,
            segments: segments,
            summaryMessage: summaryMessage,
            encouragementMessage: encouragementMessage,
            sessionFinishedAt: finishedAt,
            errorMessage: null,
            noticeMessage: noticeMessage,
            lastResolution: null,
            recitationAnalysis: recitationAnalysis,
            liveSegmentText: null,
          );
        });
        unawaited(_persistVoiceLearningSession(
          mode: mode,
          scene: scene,
          segments: segments,
          transcript: normalized,
          summaryMessage: summaryMessage,
          encouragementMessage: encouragementMessage,
          startedAt: _voiceAssistantState.sessionStartedAt ?? finishedAt,
          finishedAt: finishedAt,
          recitationAnalysis: recitationAnalysis,
        ));
        _maybeAutoSpeakVoiceWorkbenchEncouragement(encouragementMessage);
        return;
      }

      final commandRequest = request;
      final commandSurface = surface;
      if (commandRequest == null || commandSurface == null) {
        throw StateError('语音指令上下文丢失，请重新开始。');
      }

      final context = _buildVoiceCommandContext(commandSurface);
      setState(() {
        _voiceAssistantState = _voiceAssistantState.copyWith(
          isListening: false,
          isResolving: true,
          lastTranscript: normalized,
          segments: segments,
          summaryMessage: _buildVoiceSummary(
            mode: mode,
            scene: scene,
            segments: segments,
            transcript: normalized,
          ),
          encouragementMessage: _buildVoiceEncouragement(
            mode: mode,
            scene: scene,
          ),
          sessionFinishedAt: finishedAt,
          errorMessage: null,
          noticeMessage: '正在理解整段语音：$normalized',
          recitationAnalysis: null,
          liveSegmentText: null,
        );
      });

      final resolution = await widget.repository.resolveVoiceCommand(
        commandRequest,
        transcript: normalized,
        context: context,
      );
      await _applyVoiceCommandResolution(
        commandRequest,
        transcript: normalized,
        resolution: resolution,
      );
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _voiceAssistantState = _voiceAssistantState.copyWith(
          isListening: false,
          isResolving: false,
          errorMessage: '语音指令失败：$error',
          noticeMessage: null,
          liveSegmentText: null,
          sessionFinishedAt: DateTime.now(),
        );
      });
    } finally {
      _activeVoiceRequest = null;
      _activeVoiceSurface = null;
    }
  }

  VoiceCommandContext _buildVoiceCommandContext(VoiceCommandSurface surface) {
    if (surface == VoiceCommandSurface.dictation) {
      final state = _wordController.state;
      return VoiceCommandContext(
        surface: surface,
        locale: _voiceLocaleForSurface(surface),
        examples: surface.sampleUtterances,
        dictation: VoiceCommandDictationContext(
          sessionId: state.session?.sessionId,
          currentWord: state.currentWord.isEmpty ? null : state.currentWord,
          currentIndex:
              state.session?.currentIndex ?? state.currentDisplayIndex,
          totalItems: state.totalWords,
          canNext: state.canNext,
          canPrevious: state.canPrevious,
          isCompleted: state.session?.isCompleted ?? false,
          language: state.language.name,
          playbackMode: state.mode.name,
        ),
      );
    }

    final board = _controller.state.board;
    return VoiceCommandContext(
      surface: surface,
      locale: _voiceLocaleForSurface(surface),
      examples: surface.sampleUtterances,
      taskBoard: VoiceCommandTaskBoardContext(
        focusedSubject: board != null &&
                board.groups.isNotEmpty &&
                _selectedSubjectIndex < board.groups.length
            ? board.groups[_selectedSubjectIndex].subject
            : null,
        summary: VoiceCommandTaskBoardSummary(
          total: board?.summary.total ?? 0,
          completed: board?.summary.completed ?? 0,
          pending: board?.summary.pending ?? 0,
        ),
        subjects: board?.groups
                .map((group) => VoiceCommandTaskSubject(
                      subject: group.subject,
                      status: group.status,
                      completed: group.completed,
                      pending: group.pending,
                      total: group.total,
                    ))
                .toList() ??
            const <VoiceCommandTaskSubject>[],
        groups: board?.homeworkGroups
                .map((group) => VoiceCommandTaskGroup(
                      subject: group.subject,
                      groupTitle: group.groupTitle,
                      status: group.status,
                      completed: group.completed,
                      pending: group.pending,
                      total: group.total,
                    ))
                .toList() ??
            const <VoiceCommandTaskGroup>[],
        tasks: board?.tasks
                .map((task) => VoiceCommandTaskItem(
                      taskId: task.taskId,
                      subject: task.subject,
                      groupTitle: task.groupTitle,
                      content: task.content,
                      completed: task.completed,
                      status: task.status,
                    ))
                .toList() ??
            const <VoiceCommandTaskItem>[],
      ),
    );
  }

  Future<void> _applyVoiceCommandResolution(
    TaskBoardRequest request, {
    required String transcript,
    required VoiceCommandResolution resolution,
  }) async {
    final board = _controller.state.board;
    var notice = '';

    switch (resolution.action) {
      case 'dictation_next':
        await _wordController.nextWord(request.apiBaseUrl);
        notice = '已根据“$transcript”切到下一词。';
        break;
      case 'dictation_previous':
        await _wordController.previousWord(request.apiBaseUrl);
        notice = '已根据“$transcript”返回上一词。';
        break;
      case 'dictation_replay':
        await _wordController.replayCurrent(request.apiBaseUrl);
        notice = '已根据“$transcript”重播当前词。';
        break;
      case 'task_complete_item':
        if (board == null) {
          throw StateError('任务板还没有加载好。');
        }
        final task = _findTask(board, resolution.target);
        if (task == null) {
          throw StateError('没有找到要完成的任务。');
        }
        await _updateSingleTask(task, true);
        notice = '已把“${task.content}”标记为完成。';
        break;
      case 'task_complete_group':
        if (board == null) {
          throw StateError('任务板还没有加载好。');
        }
        final group = _findHomeworkGroup(board, resolution.target);
        if (group == null) {
          throw StateError('没有找到对应的任务分组。');
        }
        await _updateHomeworkGroup(group, true);
        notice = '已把“${group.groupTitle}”分组标记为完成。';
        break;
      case 'task_complete_subject':
        if (board == null) {
          throw StateError('任务板还没有加载好。');
        }
        final subjectGroup = _findSubjectGroup(board, resolution.target);
        if (subjectGroup == null) {
          throw StateError('没有找到对应的学科任务。');
        }
        await _updateSubjectGroup(subjectGroup, true);
        notice = '已把“${subjectGroup.subject}”学科任务标记为完成。';
        break;
      case 'task_complete_all':
        await _updateAllTasks(true);
        notice = '已把全部任务标记为完成。';
        break;
      case 'none':
      default:
        notice = resolution.reason.isNotEmpty
            ? '这句先不执行：${resolution.reason}'
            : '这句语音暂时没有可执行动作。';
        break;
    }

    if (!mounted) return;
    setState(() {
      _voiceAssistantState = _voiceAssistantState.copyWith(
        isListening: false,
        isResolving: false,
        errorMessage: null,
        noticeMessage: notice,
        lastTranscript: transcript,
        lastResolution: resolution,
      );
    });
  }

  TaskItem? _findTask(TaskBoard board, VoiceCommandTarget target) {
    for (final task in board.tasks) {
      if (target.taskId != null && task.taskId == target.taskId) {
        return task;
      }
    }
    for (final task in board.tasks) {
      final sameContent = (target.taskContent?.trim().isNotEmpty ?? false) &&
          task.content.trim() == target.taskContent!.trim();
      final sameSubject = (target.subject?.trim().isEmpty ?? true) ||
          task.subject.trim() == target.subject!.trim();
      if (sameContent && sameSubject) {
        return task;
      }
    }
    return null;
  }

  HomeworkGroup? _findHomeworkGroup(
      TaskBoard board, VoiceCommandTarget target) {
    for (final group in board.homeworkGroups) {
      final sameSubject = (target.subject?.trim().isEmpty ?? true) ||
          group.subject.trim() == target.subject!.trim();
      final sameGroupTitle = (target.groupTitle?.trim().isNotEmpty ?? false) &&
          group.groupTitle.trim() == target.groupTitle!.trim();
      if (sameSubject && sameGroupTitle) {
        return group;
      }
    }
    return null;
  }

  TaskGroup? _findSubjectGroup(TaskBoard board, VoiceCommandTarget target) {
    for (final group in board.groups) {
      if ((target.subject?.trim().isNotEmpty ?? false) &&
          group.subject.trim() == target.subject!.trim()) {
        return group;
      }
    }
    return null;
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
        listenable: _controller,
        builder: (context, _) {
          final state = _controller.state;
          final voiceSurface = _currentVoiceSurface;
          return Scaffold(
            appBar: AppBar(title: const Text('StudyClaw Pad'), actions: [
              IconButton(
                  onPressed: _refreshBoard,
                  icon: const Icon(Icons.refresh_rounded)),
              IconButton(
                  onPressed: _openSettingsSheet,
                  icon: const Icon(Icons.tune_rounded)),
              const SizedBox(width: 8)
            ]),
            body: SafeArea(
                child: Stack(children: [
              RefreshIndicator(
                  onRefresh: _refreshBoard,
                  child: ListView(padding: const EdgeInsets.all(24), children: [
                    _TodayHeroCard(
                        date: _dateController.text.trim(),
                        onPreviousDate: () => _shiftDate(-1),
                        onNextDate: () => _shiftDate(1),
                        onStartRecommended: _controller.hotTaskLaunchEnabled
                            ? () => _startRecommendedTask()
                            : null,
                        showStartRecommended:
                            _controller.hotTaskLaunchEnabled,
                        onCompleteAll: () => _updateAllTasks(true),
                        onResetAll: () => _updateAllTasks(false)),
                    if (state.errorMessage != null) ...[
                      const SizedBox(height: 16),
                      _BannerCard(message: state.errorMessage!)
                    ],
                    const SizedBox(height: 24),
                    _HomeModeSwitcher(
                        selectedTab: _selectedTab, onChanged: _setSelectedTab),
                    const SizedBox(height: 24),
                    _VoiceAssistantCard(
                      key: const Key('voice-assistant-card'),
                      surface: voiceSurface,
                      state: _voiceAssistantState,
                      supportsRecognition:
                          _speechRecognizer.supportsRecognition,
                      supportsEncouragementVoice:
                          _wordController.supportsPlayback,
                      encouragementVoiceEnabled: _encouragementVoiceEnabled,
                      availableScenes: _availableVoiceScenes(
                        _voiceAssistantState.mode,
                        _selectedTab,
                      ),
                      sessionDuration:
                          _voiceSessionDuration(_voiceAssistantState),
                      hintText: _voiceHintForState(_voiceAssistantState),
                      sampleUtterances:
                          _voiceSamplesForState(_voiceAssistantState),
                      referenceController: _voiceReferenceController,
                      showReferenceInput:
                          _showManualReferenceInput(_voiceAssistantState),
                      referenceSummaryTitle:
                          _taskReferenceSummaryTitle(_voiceAssistantState),
                      referenceSummaryDetail:
                          _taskReferenceSummaryDetail(_voiceAssistantState),
                      referenceLabel:
                          _referenceInputLabel(_voiceAssistantState),
                      referenceHint: _referenceInputHint(_voiceAssistantState),
                      onModeChanged: _setVoiceWorkbenchMode,
                      onSceneChanged: _setVoiceLearningScene,
                      onClear: _clearVoiceWorkbench,
                      onReplayEncouragementVoice:
                          _replayVoiceWorkbenchEncouragement,
                      onToggleEncouragementVoice: _toggleEncouragementVoice,
                      onTrigger: _voiceAssistantState.isResolving
                          ? null
                          : _toggleVoiceAssistant,
                    ),
                    const SizedBox(height: 24),
                    if (_selectedTab == _PadHomeTab.tasks) ...[
                      if (_taskEncouragementMessage(state) != null) ...[
                        _TaskEncouragementCard(
                          key: const Key('task-encouragement-card'),
                          message: _taskEncouragementMessage(state)!,
                          totals: state.dailyStats?.totals,
                          tone: state.noticeTone,
                          supportsVoice: _wordController.supportsPlayback,
                          voiceEnabled: _encouragementVoiceEnabled,
                          onReplayVoice: _replayTaskEncouragement,
                          onToggleVoice: _toggleEncouragementVoice,
                        ),
                        const SizedBox(height: 24),
                      ],
                      ..._buildBoardSections(state)
                    ] else
                      ListenableBuilder(
                        listenable: _wordController,
                        builder: (context, _) {
                          final wState = _wordController.state;
                          return _WordPlaybackPanel(
                            state: wState,
                            onLanguageChanged: _wordController.setLanguage,
                            onModeChanged: _wordController.setMode,
                            onPeekChanged: _wordController.setPeeking,
                            onSyncBackend: () {
                              final r = _buildRequest();
                              if (r != null) _wordController.syncWordList(r);
                            },
                            onPlayCurrent: () async {
                              if (_wordController.state.session == null) {
                                final r = _buildRequest();
                                if (r != null) {
                                  await _wordController.startDictation(r);
                                  return;
                                }
                              }
                              await _wordController.playCurrent();
                            },
                            onReplayCurrent: () => _wordController
                                .replayCurrent(_apiBaseUrlController.text),
                            onNextWord: () => _wordController
                                .nextWord(_apiBaseUrlController.text),
                            onPreviousWord: () => _wordController
                                .previousWord(_apiBaseUrlController.text),
                            onSubmitPhoto: _takeAndSubmitPhoto,
                          );
                        },
                      ),
                  ])),
              Align(
                  alignment: Alignment.center,
                  child: IgnorePointer(
                      child: AnimatedBuilder(
                          animation: _celebrationController,
                          builder: (context, _) {
                            if (_celebrationController.value == 0) {
                              return const SizedBox.shrink();
                            }
                            return Opacity(
                                opacity: (1.0 - _celebrationController.value)
                                    .clamp(0.0, 1.0),
                                child: Transform.scale(
                                    scale: 1.0 +
                                        _celebrationController.value * 2.0,
                                    child: const Icon(Icons.stars_rounded,
                                        color: KidColors.color4, size: 160)));
                          }))),
            ])),
          );
        });
  }

  void _shiftDate(int d) {
    _dateController.text = formatTaskBoardDate(
        (parseTaskBoardDate(_dateController.text.trim()) ?? DateTime.now())
            .add(Duration(days: d)));
    _loadBoard(showLoadingState: true);
  }

  String? _taskEncouragementMessage(TaskBoardViewState state) {
    final notice = state.noticeMessage?.trim() ?? '';
    if (notice.isNotEmpty) {
      return notice;
    }

    final encouragement = state.dailyStats?.encouragement.trim() ?? '';
    if (encouragement.isNotEmpty) {
      return encouragement;
    }
    return null;
  }

  List<Widget> _buildBoardSections(TaskBoardViewState state) {
    final board = state.board;
    if (board == null) {
      return [const KidInlineLoading(title: '同步中', description: '正在准备挑战舞台...')];
    }
    final groups = board.groups;
    if (groups.isEmpty) {
      return [const _EmptyBoard(title: '暂无任务', description: '表现真棒！')];
    }
    if (_selectedSubjectIndex >= groups.length) {
      _selectedSubjectIndex = 0;
    }
    final focused = groups[_selectedSubjectIndex];
    return [
      _SubjectNavigator(
          groups: groups,
          selectedIndex: _selectedSubjectIndex,
          onSelected: (i) => setState(() => _selectedSubjectIndex = i)),
      const SizedBox(height: 24),
      _FocusedSubjectStage(
          group: focused,
          homeworkGroups: board.homeworkGroups
              .where((h) => h.subject == focused.subject)
              .toList(),
          tasks:
              board.tasks.where((t) => t.subject == focused.subject).toList(),
          busy: state.isBusy,
          onToggleTask: _updateSingleTask),
    ];
  }
}

class _TaskEncouragementCard extends StatelessWidget {
  const _TaskEncouragementCard({
    super.key,
    required this.message,
    required this.totals,
    required this.tone,
    required this.supportsVoice,
    required this.voiceEnabled,
    required this.onReplayVoice,
    required this.onToggleVoice,
  });

  final String message;
  final StatsTotals? totals;
  final TaskBoardNoticeTone tone;
  final bool supportsVoice;
  final bool voiceEnabled;
  final Future<void> Function() onReplayVoice;
  final VoidCallback onToggleVoice;

  @override
  Widget build(BuildContext context) {
    final accentColor =
        tone == TaskBoardNoticeTone.info ? KidColors.color2 : KidColors.color3;
    final summaryChips = <Widget>[
      if (totals != null && totals!.totalTasks > 0)
        _MiniTraceChip(
            label: '已完成 ${totals!.completedTasks}/${totals!.totalTasks}'),
      if (totals != null && totals!.pendingTasks > 0)
        _MiniTraceChip(label: '剩余 ${totals!.pendingTasks} 项'),
      if (totals != null && totals!.totalPointsDelta != 0)
        _MiniTraceChip(label: '积分 ${totals!.pointsDeltaLabel}'),
    ];

    return KidCard(
      borderColor: KidColors.black,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 42,
                height: 42,
                decoration: BoxDecoration(
                  color: accentColor.withAlpha(40),
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.favorite_rounded,
                  color: accentColor,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  '成长小鼓励',
                  style: TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.w900,
                    color: accentColor,
                  ),
                ),
              ),
            ],
          ),
          if (supportsVoice) ...[
            const SizedBox(height: 12),
            Align(
              alignment: Alignment.centerRight,
              child: _EncouragementVoiceControls(
                replayKey: const Key('task-encouragement-replay'),
                toggleKey: const Key('task-encouragement-voice-toggle'),
                voiceEnabled: voiceEnabled,
                onReplay: () {
                  unawaited(onReplayVoice());
                },
                onToggle: onToggleVoice,
              ),
            ),
          ],
          const SizedBox(height: 16),
          Text(
            message,
            style: const TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w800,
              height: 1.5,
            ),
          ),
          if (summaryChips.isNotEmpty) ...[
            const SizedBox(height: 16),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: summaryChips,
            ),
          ],
        ],
      ),
    );
  }
}

class _TodayHeroCard extends StatelessWidget {
  const _TodayHeroCard(
      {required this.date,
      required this.onPreviousDate,
      required this.onNextDate,
      required this.onStartRecommended,
      required this.showStartRecommended,
      required this.onCompleteAll,
      required this.onResetAll});
  final String date;
  final VoidCallback? onPreviousDate,
      onNextDate,
      onStartRecommended,
      onCompleteAll,
      onResetAll;
  final bool showStartRecommended;
  @override
  Widget build(BuildContext context) => KidCard(
      color: KidColors.color1,
      hasBorder: false,
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Expanded(
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                Text(date,
                    style: const TextStyle(
                        color: KidColors.white,
                        fontSize: 16,
                        fontWeight: FontWeight.w700)),
                const Text('挑战舞台',
                    style: TextStyle(
                        color: KidColors.white,
                        fontSize: 32,
                        fontWeight: FontWeight.w900))
              ])),
          const Icon(Icons.rocket_launch_rounded,
              color: KidColors.white, size: 40)
        ]),
        const SizedBox(height: 24),
        Row(children: [
          IconButton(
              onPressed: onPreviousDate,
              icon:
                  const Icon(Icons.chevron_left_rounded, color: Colors.white)),
          const Spacer(),
          const Text('前进！向终点冲刺',
              style:
                  TextStyle(color: Colors.white, fontWeight: FontWeight.w800)),
          const Spacer(),
          IconButton(
              onPressed: onNextDate,
              icon:
                  const Icon(Icons.chevron_right_rounded, color: Colors.white))
        ]),
        const SizedBox(height: 24),
        Row(children: [
          if (showStartRecommended) ...[
            Expanded(
                child: KidSmallBtn(
                    label: '先做推荐',
                    color: KidColors.color4,
                    onTap: onStartRecommended)),
            const SizedBox(width: 12),
          ],
          Expanded(
              child: KidSmallBtn(
                  label: '全部重置', color: KidColors.color5, onTap: onResetAll)),
          const SizedBox(width: 12),
          Expanded(
              child: KidSmallBtn(
                  label: '一键完成', color: KidColors.color3, onTap: onCompleteAll))
        ]),
      ]));
}

class _SubjectNavigator extends StatelessWidget {
  const _SubjectNavigator(
      {required this.groups,
      required this.selectedIndex,
      required this.onSelected});
  final List<TaskGroup> groups;
  final int selectedIndex;
  final ValueChanged<int> onSelected;
  @override
  Widget build(BuildContext context) => SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
          children: List.generate(groups.length, (i) {
        final g = groups[i];
        final isSel = i == selectedIndex;
        final color = g.subject.contains('语')
            ? KidColors.color5
            : (g.subject.contains('数') ? KidColors.color2 : KidColors.color4);
        return GestureDetector(
            onTap: () => onSelected(i),
            child: Container(
                margin: const EdgeInsets.only(right: 12),
                padding:
                    const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
                decoration: BoxDecoration(
                    color: isSel ? color : KidColors.white,
                    borderRadius: BorderRadius.circular(20),
                    border: Border.all(
                        color: isSel ? color : KidColors.black, width: 2)),
                child: Row(children: [
                  if (g.status == 'completed')
                    const Icon(Icons.check_circle_rounded,
                        color: KidColors.color3, size: 18),
                  const SizedBox(width: 4),
                  Text(g.subject,
                      style: TextStyle(
                          fontWeight: FontWeight.w900,
                          color: isSel ? KidColors.white : KidColors.black)),
                  const SizedBox(width: 8),
                  Text('${g.completed}/${g.total}',
                      style: TextStyle(
                          fontWeight: FontWeight.w900,
                          color: isSel ? KidColors.white : KidColors.black))
                ])));
      })));
}

class _FocusedSubjectStage extends StatelessWidget {
  const _FocusedSubjectStage(
      {required this.group,
      required this.homeworkGroups,
      required this.tasks,
      required this.busy,
      required this.onToggleTask});
  final TaskGroup group;
  final List<HomeworkGroup> homeworkGroups;
  final List<TaskItem> tasks;
  final bool busy;
  final Future<void> Function(TaskItem, bool) onToggleTask;
  @override
  Widget build(BuildContext context) => KidCard(
      borderColor: KidColors.black,
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('${group.subject} 挑战舞台',
            style: const TextStyle(fontSize: 24, fontWeight: FontWeight.w900)),
        const SizedBox(height: 20),
        ...homeworkGroups.map(
            (hw) =>
                Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Padding(
                      padding: const EdgeInsets.symmetric(vertical: 8),
                      child: Text(hw.groupTitle,
                          style: const TextStyle(
                              fontSize: 18, fontWeight: FontWeight.w800))),
                  ...tasks.where((t) => t.groupTitle == hw.groupTitle).map(
                      (t) => Container(
                          margin: const EdgeInsets.only(bottom: 8),
                          decoration: BoxDecoration(
                              color: t.completed
                                  ? KidColors.color3.withAlpha(30)
                                  : Colors.white,
                              borderRadius: BorderRadius.circular(16),
                              border: Border.all(
                                  color: t.completed
                                      ? KidColors.color3
                                      : KidColors.black,
                                  width: 2)),
                          child: CheckboxListTile(
                              value: t.completed,
                              onChanged: (v) => onToggleTask(t, v ?? false),
                              activeColor: KidColors.color3,
                              title: Text(t.content,
                                  style: TextStyle(
                                      fontWeight: FontWeight.w700,
                                      decoration: t.completed
                                          ? TextDecoration.lineThrough
                                          : null))))),
                ])),
      ]));
}

class _VoiceAssistantCard extends StatelessWidget {
  const _VoiceAssistantCard({
    super.key,
    required this.surface,
    required this.state,
    required this.supportsRecognition,
    required this.supportsEncouragementVoice,
    required this.encouragementVoiceEnabled,
    required this.availableScenes,
    required this.sessionDuration,
    required this.hintText,
    required this.sampleUtterances,
    required this.referenceController,
    required this.showReferenceInput,
    required this.referenceSummaryTitle,
    required this.referenceSummaryDetail,
    required this.referenceLabel,
    required this.referenceHint,
    required this.onModeChanged,
    required this.onSceneChanged,
    required this.onClear,
    required this.onReplayEncouragementVoice,
    required this.onToggleEncouragementVoice,
    required this.onTrigger,
  });

  final VoiceCommandSurface surface;
  final _VoiceAssistantState state;
  final bool supportsRecognition;
  final bool supportsEncouragementVoice;
  final bool encouragementVoiceEnabled;
  final List<_LearningScene> availableScenes;
  final Duration sessionDuration;
  final String hintText;
  final List<String> sampleUtterances;
  final TextEditingController referenceController;
  final bool showReferenceInput;
  final String? referenceSummaryTitle;
  final String? referenceSummaryDetail;
  final String referenceLabel;
  final String referenceHint;
  final ValueChanged<_SpeechWorkbenchMode> onModeChanged;
  final ValueChanged<_LearningScene> onSceneChanged;
  final VoidCallback? onClear;
  final Future<void> Function() onReplayEncouragementVoice;
  final VoidCallback onToggleEncouragementVoice;
  final VoidCallback? onTrigger;

  @override
  Widget build(BuildContext context) {
    final buttonLabel = state.isListening
        ? '结束说话'
        : state.isResolving
            ? '理解中'
            : '开始说话';
    final metricTiles = <Widget>[
      _VoiceWorkbenchStatTile(
        label: '模式',
        value: state.mode.label,
        color: state.mode.accentColor,
      ),
      _VoiceWorkbenchStatTile(
        label: '场景',
        value: state.scene.label,
        color: KidColors.color1,
      ),
      _VoiceWorkbenchStatTile(
        label: '时长',
        value: _formatVoiceClock(sessionDuration),
        color: KidColors.color4,
      ),
      _VoiceWorkbenchStatTile(
        label: '分段',
        value: '${state.segments.length}',
        color: KidColors.color3,
      ),
    ];

    return KidCard(
      borderColor: KidColors.black,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.mic_rounded, size: 28),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      '孩子学习语音工作台',
                      style:
                          TextStyle(fontSize: 22, fontWeight: FontWeight.w900),
                    ),
                    Text(
                      '入口 ${surface.label} · ${state.mode.label} · ${state.scene.label}',
                      style: TextStyle(
                        fontWeight: FontWeight.w800,
                        color: KidColors.black.withAlpha(170),
                      ),
                    ),
                  ],
                ),
              ),
              SizedBox(
                width: 140,
                child: KidSmallBtn(
                  key: const Key('voice-assistant-trigger'),
                  label: buttonLabel,
                  color: state.isListening
                      ? KidColors.color5
                      : state.mode.accentColor,
                  onTap: onTrigger,
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Text(
            '一张面板切换三种语音模式：短指令、长段记录、陪伴监听。',
            style: TextStyle(
              fontWeight: FontWeight.w800,
              color: KidColors.black.withAlpha(170),
            ),
          ),
          const SizedBox(height: 16),
          Wrap(
            spacing: 10,
            runSpacing: 10,
            children: _SpeechWorkbenchMode.values.map((mode) {
              final selected = mode == state.mode;
              return _VoiceWorkbenchChoiceChip(
                key: Key('voice-mode-${mode.name}'),
                icon: mode.icon,
                label: mode.label,
                caption: mode.description,
                color: mode.accentColor,
                selected: selected,
                onTap: state.isBusy ? null : () => onModeChanged(mode),
              );
            }).toList(),
          ),
          const SizedBox(height: 16),
          Text(
            '学习场景',
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w900,
              color: KidColors.black.withAlpha(220),
            ),
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 10,
            runSpacing: 10,
            children: availableScenes.map((scene) {
              final selected = scene == state.scene;
              return _VoiceWorkbenchChoiceChip(
                key: Key('voice-scene-${scene.name}'),
                icon: Icons.auto_stories_rounded,
                label: scene.label,
                caption: scene.description,
                color: selected
                    ? state.mode.accentColor
                    : KidColors.black.withAlpha(140),
                selected: selected,
                compact: true,
                onTap: state.isBusy ? null : () => onSceneChanged(scene),
              );
            }).toList(),
          ),
          const SizedBox(height: 16),
          Text(
            hintText,
            style: TextStyle(
              fontWeight: FontWeight.w800,
              color: KidColors.black.withAlpha(170),
            ),
          ),
          const SizedBox(height: 14),
          Wrap(
            spacing: 12,
            runSpacing: 12,
            children: metricTiles,
          ),
          const SizedBox(height: 16),
          Text(
            '示例话术',
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w900,
              color: KidColors.black.withAlpha(220),
            ),
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: sampleUtterances
                .map((item) => _MiniTraceChip(label: item))
                .toList(),
          ),
          if (referenceSummaryTitle != null &&
              referenceSummaryDetail != null) ...[
            const SizedBox(height: 18),
            _VoiceWorkbenchSectionCard(
              key: const Key('voice-reference-task-summary'),
              title: referenceSummaryTitle!,
              icon: Icons.lock_outline_rounded,
              accentColor: KidColors.color3,
              child: Text(
                referenceSummaryDetail!,
                style: TextStyle(
                  fontWeight: FontWeight.w800,
                  color: KidColors.black.withAlpha(170),
                  height: 1.45,
                ),
              ),
            ),
          ] else if (showReferenceInput) ...[
            const SizedBox(height: 18),
            _VoiceWorkbenchSectionCard(
              title: '$referenceLabel参考台',
              icon: Icons.menu_book_rounded,
              accentColor: KidColors.color2,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    referenceHint,
                    style: TextStyle(
                      fontWeight: FontWeight.w700,
                      color: KidColors.black.withAlpha(170),
                      height: 1.45,
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    key: const Key('voice-reference-input'),
                    controller: referenceController,
                    minLines: 4,
                    maxLines: 8,
                    enabled: !state.isBusy,
                    decoration: InputDecoration(
                      hintText:
                          '例如：江畔独步寻花【唐】杜甫\n黄师塔前江水东，春光懒困倚微风。\n桃花一簇开无主，可爱深红爱浅红？',
                      filled: true,
                      fillColor: KidColors.white,
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(18),
                      ),
                      enabledBorder: OutlineInputBorder(
                        borderRadius: BorderRadius.circular(18),
                        borderSide: BorderSide(
                          color: KidColors.black.withAlpha(120),
                          width: 2,
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
          const SizedBox(height: 18),
          _VoiceWorkbenchSectionCard(
            title: '实时记录',
            icon: state.isListening ? Icons.graphic_eq_rounded : Icons.notes,
            accentColor: state.mode.accentColor,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                if (state.lastTranscript != null)
                  Text(
                    state.isListening
                        ? '正在记录：${state.lastTranscript}'
                        : '刚刚听到：${state.lastTranscript}',
                    style: const TextStyle(fontWeight: FontWeight.w900),
                  )
                else
                  Text(
                    supportsRecognition
                        ? '开始说话后，这里会像学习记录板一样实时出现转写内容。'
                        : '当前设备不支持语音识别，暂时无法开始记录。',
                    style: TextStyle(
                      fontWeight: FontWeight.w700,
                      color: KidColors.black.withAlpha(170),
                    ),
                  ),
                if (state.segments.isNotEmpty) ...[
                  const SizedBox(height: 14),
                  ...state.segments.map((segment) {
                    return Container(
                      margin: const EdgeInsets.only(bottom: 10),
                      padding: const EdgeInsets.all(14),
                      decoration: BoxDecoration(
                        color: segment.isLive
                            ? state.mode.accentColor.withAlpha(24)
                            : KidColors.white,
                        borderRadius: BorderRadius.circular(18),
                        border: Border.all(
                          color: segment.isLive
                              ? state.mode.accentColor
                              : KidColors.black.withAlpha(90),
                          width: 2,
                        ),
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            children: [
                              Text(
                                '第 ${segment.index} 段',
                                style: const TextStyle(
                                  fontWeight: FontWeight.w900,
                                ),
                              ),
                              const SizedBox(width: 8),
                              Text(
                                _formatVoiceSegmentTime(segment.capturedAt),
                                style: TextStyle(
                                  fontWeight: FontWeight.w700,
                                  color: KidColors.black.withAlpha(140),
                                ),
                              ),
                              const Spacer(),
                              if (segment.isLive)
                                Container(
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 10,
                                    vertical: 6,
                                  ),
                                  decoration: BoxDecoration(
                                    color: state.mode.accentColor.withAlpha(36),
                                    borderRadius: BorderRadius.circular(999),
                                  ),
                                  child: Text(
                                    '进行中',
                                    style: TextStyle(
                                      fontWeight: FontWeight.w900,
                                      color: state.mode.accentColor,
                                    ),
                                  ),
                                ),
                            ],
                          ),
                          const SizedBox(height: 8),
                          Text(
                            segment.text,
                            style: const TextStyle(
                              fontSize: 16,
                              fontWeight: FontWeight.w700,
                              height: 1.45,
                            ),
                          ),
                        ],
                      ),
                    );
                  }),
                ],
              ],
            ),
          ),
          if (state.summaryMessage != null) ...[
            const SizedBox(height: 16),
            _VoiceWorkbenchSectionCard(
              key: const Key('voice-workbench-summary'),
              title: '整理结果',
              icon: Icons.insights_rounded,
              accentColor: KidColors.color1,
              child: Text(
                state.summaryMessage!,
                style: const TextStyle(
                  fontWeight: FontWeight.w800,
                  height: 1.5,
                ),
              ),
            ),
          ],
          if (state.encouragementMessage != null) ...[
            const SizedBox(height: 16),
            _VoiceWorkbenchSectionCard(
              title: '成长鼓励',
              icon: Icons.star_rounded,
              accentColor: KidColors.color3,
              headerAction: supportsEncouragementVoice
                  ? _EncouragementVoiceControls(
                      replayKey: const Key('voice-encouragement-replay'),
                      toggleKey: const Key('voice-encouragement-voice-toggle'),
                      voiceEnabled: encouragementVoiceEnabled,
                      onReplay: () {
                        unawaited(onReplayEncouragementVoice());
                      },
                      onToggle: onToggleEncouragementVoice,
                    )
                  : null,
              child: Text(
                state.encouragementMessage!,
                style: const TextStyle(
                  fontWeight: FontWeight.w900,
                  height: 1.5,
                ),
              ),
            ),
          ],
          if (state.recitationAnalysis != null) ...[
            const SizedBox(height: 16),
            _VoiceWorkbenchSectionCard(
              key: const Key('voice-recitation-analysis'),
              title: '背诵对照',
              icon: Icons.auto_awesome_rounded,
              accentColor: state.recitationAnalysis!.needsRetry
                  ? KidColors.color5
                  : KidColors.color3,
              child: _RecitationAnalysisPanel(
                analysis: state.recitationAnalysis!,
              ),
            ),
          ],
          if (state.lastResolution != null) ...[
            const SizedBox(height: 12),
            Text(
              '理解结果：${state.lastResolution!.actionLabel}',
              style: TextStyle(
                fontWeight: FontWeight.w900,
                color: KidColors.color1.withAlpha(210),
              ),
            ),
            if (state.lastResolution!.reason.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(
                  state.lastResolution!.reason,
                  style: TextStyle(
                    fontWeight: FontWeight.w700,
                    color: KidColors.black.withAlpha(170),
                  ),
                ),
              ),
          ],
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(
                child: KidSmallBtn(
                  key: const Key('voice-assistant-clear'),
                  label: '清空记录',
                  color: KidColors.color4,
                  onTap: onClear,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: KidSmallBtn(
                  label: state.mode == _SpeechWorkbenchMode.command
                      ? '短句执行'
                      : state.mode == _SpeechWorkbenchMode.transcript
                          ? '整理整段'
                          : '陪伴收听',
                  color: state.mode.accentColor,
                  onTap: onTrigger,
                ),
              ),
            ],
          ),
          if (state.noticeMessage != null || state.errorMessage != null) ...[
            const SizedBox(height: 16),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: state.errorMessage != null
                    ? KidColors.color5.withAlpha(34)
                    : KidColors.color3.withAlpha(30),
                borderRadius: BorderRadius.circular(20),
                border: Border.all(
                  color: state.errorMessage != null
                      ? KidColors.color5
                      : KidColors.color3,
                  width: 2,
                ),
              ),
              child: Text(
                state.errorMessage ?? state.noticeMessage!,
                style: TextStyle(
                  fontWeight: FontWeight.w800,
                  color: state.errorMessage != null
                      ? KidColors.color5
                      : KidColors.color3.withAlpha(220),
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _VoiceWorkbenchChoiceChip extends StatelessWidget {
  const _VoiceWorkbenchChoiceChip({
    super.key,
    required this.icon,
    required this.label,
    required this.caption,
    required this.color,
    required this.selected,
    this.compact = false,
    this.onTap,
  });

  final IconData icon;
  final String label;
  final String caption;
  final Color color;
  final bool selected;
  final bool compact;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final backgroundColor =
        selected ? color.withAlpha(36) : KidColors.white.withAlpha(240);
    final borderColor =
        selected ? color : KidColors.black.withAlpha(onTap == null ? 50 : 130);

    return GestureDetector(
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 180),
        padding: EdgeInsets.symmetric(
          horizontal: compact ? 14 : 16,
          vertical: compact ? 12 : 14,
        ),
        width: compact ? 168 : 210,
        decoration: BoxDecoration(
          color: backgroundColor,
          borderRadius: BorderRadius.circular(20),
          border: Border.all(color: borderColor, width: 2),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              width: compact ? 34 : 38,
              height: compact ? 34 : 38,
              decoration: BoxDecoration(
                color: color.withAlpha(selected ? 42 : 24),
                shape: BoxShape.circle,
              ),
              child: Icon(icon, color: color, size: compact ? 18 : 20),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    label,
                    style: TextStyle(
                      fontWeight: FontWeight.w900,
                      color: selected ? color : KidColors.black,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    caption,
                    maxLines: compact ? 2 : 3,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: compact ? 12 : 13,
                      fontWeight: FontWeight.w700,
                      color: KidColors.black.withAlpha(160),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _VoiceWorkbenchStatTile extends StatelessWidget {
  const _VoiceWorkbenchStatTile({
    required this.label,
    required this.value,
    required this.color,
  });

  final String label;
  final String value;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 112,
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: color.withAlpha(26),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: color, width: 2),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: TextStyle(
              fontWeight: FontWeight.w800,
              color: color,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            value,
            style: const TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w900,
            ),
          ),
        ],
      ),
    );
  }
}

class _EncouragementVoiceControls extends StatelessWidget {
  const _EncouragementVoiceControls({
    required this.replayKey,
    required this.toggleKey,
    required this.voiceEnabled,
    required this.onReplay,
    required this.onToggle,
  });

  final Key replayKey;
  final Key toggleKey;
  final bool voiceEnabled;
  final VoidCallback onReplay;
  final VoidCallback onToggle;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      alignment: WrapAlignment.end,
      children: [
        _EncouragementVoiceAction(
          key: replayKey,
          icon: Icons.volume_up_rounded,
          label: '重播鼓励',
          color: KidColors.color3,
          onTap: onReplay,
        ),
        _EncouragementVoiceAction(
          key: toggleKey,
          icon: voiceEnabled
              ? Icons.record_voice_over_rounded
              : Icons.volume_off_rounded,
          label: voiceEnabled ? '自动播报开' : '自动播报关',
          color: voiceEnabled ? KidColors.color2 : KidColors.black,
          onTap: onToggle,
        ),
      ],
    );
  }
}

class _EncouragementVoiceAction extends StatelessWidget {
  const _EncouragementVoiceAction({
    super.key,
    required this.icon,
    required this.label,
    required this.color,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final Color color;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: color.withAlpha(20),
          borderRadius: BorderRadius.circular(999),
          border: Border.all(color: color, width: 2),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 18, color: color),
            const SizedBox(width: 6),
            Text(
              label,
              style: TextStyle(
                fontWeight: FontWeight.w900,
                color: color,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _VoiceWorkbenchSectionCard extends StatelessWidget {
  const _VoiceWorkbenchSectionCard({
    super.key,
    required this.title,
    required this.icon,
    required this.accentColor,
    required this.child,
    this.headerAction,
  });

  final String title;
  final IconData icon;
  final Color accentColor;
  final Widget child;
  final Widget? headerAction;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: accentColor.withAlpha(18),
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: accentColor.withAlpha(180), width: 2),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, color: accentColor),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  title,
                  style: TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.w900,
                    color: accentColor,
                  ),
                ),
              ),
            ],
          ),
          if (headerAction != null) ...[
            const SizedBox(height: 12),
            Align(
              alignment: Alignment.centerRight,
              child: headerAction!,
            ),
          ],
          const SizedBox(height: 12),
          child,
        ],
      ),
    );
  }
}

class _RecitationAnalysisPanel extends StatelessWidget {
  const _RecitationAnalysisPanel({
    required this.analysis,
  });

  final RecitationAnalysis analysis;

  @override
  Widget build(BuildContext context) {
    final issueChips = analysis.issues
        .map((item) => _MiniTraceChip(label: item))
        .toList(growable: false);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Wrap(
          spacing: 10,
          runSpacing: 10,
          children: [
            _VoiceWorkbenchStatTile(
              label: '标题',
              value: analysis.displayTitle,
              color: KidColors.color1,
            ),
            _VoiceWorkbenchStatTile(
              label: '作者',
              value: analysis.displayAuthor,
              color: KidColors.color2,
            ),
            _VoiceWorkbenchStatTile(
              label: '完成度',
              value: analysis.completionLabel,
              color: analysis.needsRetry ? KidColors.color5 : KidColors.color3,
            ),
            _VoiceWorkbenchStatTile(
              label: '分析',
              value: analysis.parserModeLabel,
              color: KidColors.color4,
            ),
          ],
        ),
        const SizedBox(height: 16),
        Text(
          analysis.summary,
          style: const TextStyle(
            fontWeight: FontWeight.w900,
            height: 1.45,
          ),
        ),
        const SizedBox(height: 10),
        Text(
          analysis.suggestion,
          style: TextStyle(
            fontWeight: FontWeight.w700,
            color: KidColors.black.withAlpha(170),
            height: 1.45,
          ),
        ),
        const SizedBox(height: 10),
        Text(
          '上面的学习记录按真实停顿保留；下面这部分按标准原文逐句对照，方便同时看“真实开口过程”和“标准完成度”。',
          style: TextStyle(
            fontWeight: FontWeight.w700,
            color: KidColors.black.withAlpha(150),
            height: 1.45,
          ),
        ),
        if (issueChips.isNotEmpty) ...[
          const SizedBox(height: 14),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: issueChips,
          ),
        ],
        if (analysis.matchedLines.isNotEmpty) ...[
          const SizedBox(height: 16),
          ...analysis.matchedLines.map((line) {
            return _RecitationLineTile(
              line: line,
              concealExpectedText: analysis.scene == 'recitation',
            );
          }),
        ],
      ],
    );
  }
}

class _RecitationLineTile extends StatelessWidget {
  const _RecitationLineTile({
    required this.line,
    required this.concealExpectedText,
  });

  final RecitationLineAnalysis line;
  final bool concealExpectedText;

  Color get _accentColor {
    if (line.isMatched) {
      return KidColors.color3;
    }
    if (line.isMissing) {
      return KidColors.color5;
    }
    return KidColors.color4;
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: _accentColor.withAlpha(18),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: _accentColor, width: 2),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Text(
                '第 ${line.index} 句',
                style: TextStyle(
                  fontWeight: FontWeight.w900,
                  color: _accentColor,
                ),
              ),
              const SizedBox(width: 8),
              _MiniTraceChip(label: line.statusLabel),
              const SizedBox(width: 8),
              _MiniTraceChip(label: line.ratioLabel),
            ],
          ),
          const SizedBox(height: 10),
          Text(
            concealExpectedText
                ? '原文提示：${line.isMatched ? '这一句基本对上' : '需要家长/老师侧查看或按提示重背'}'
                : '原文：${line.expected}',
            style: const TextStyle(
              fontWeight: FontWeight.w900,
              height: 1.45,
            ),
          ),
          const SizedBox(height: 6),
          Text(
            line.observed.trim().isEmpty
                ? '听到：这一句没有稳定识别出来'
                : '听到：${line.observed}',
            style: TextStyle(
              fontWeight: FontWeight.w700,
              color: KidColors.black.withAlpha(180),
              height: 1.45,
            ),
          ),
          if (line.notes.trim().isNotEmpty) ...[
            const SizedBox(height: 8),
            Text(
              line.notes,
              style: TextStyle(
                fontWeight: FontWeight.w700,
                color: _accentColor,
                height: 1.4,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

String _formatScoreSummary(DictationGradingResult result) {
  if (result.incorrectCount <= 0) {
    return '本次共 ${result.gradedItems.length} 个词，全对，得分 ${result.score}。';
  }
  return '本次共 ${result.gradedItems.length} 个词，错 ${result.incorrectCount} 个，得分 ${result.score}。';
}

String _formatWordCorrectionComment(DictationGradedItem item) {
  final comment = item.comment?.trim() ?? '';
  if (comment.isNotEmpty) {
    return comment;
  }
  if (item.actual.trim().isEmpty) {
    return '这一处没有识别清楚，订正时写完整一点。';
  }
  return '对照正确写法再认真订正一次。';
}

void _showAnnotatedResultPreview(
  BuildContext context,
  DictationGradingResult result,
) {
  showDialog<void>(
    context: context,
    builder: (context) => Dialog(
      insetPadding: const EdgeInsets.all(20),
      child: Container(
        constraints: const BoxConstraints(maxWidth: 720, maxHeight: 820),
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Expanded(
                  child: Text(
                    '错词标注大图',
                    style: TextStyle(fontSize: 20, fontWeight: FontWeight.w900),
                  ),
                ),
                IconButton(
                  onPressed: () => Navigator.of(context).pop(),
                  icon: const Icon(Icons.close_rounded),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Expanded(
              child: InteractiveViewer(
                minScale: 1,
                maxScale: 4,
                child: ClipRRect(
                  borderRadius: BorderRadius.circular(20),
                  child: Stack(
                    children: [
                      Container(
                        width: double.infinity,
                        color: Colors.white,
                        child: result.hasAnnotatedPhoto
                            ? Image.network(
                                result.annotatedPhotoUrl!,
                                fit: BoxFit.contain,
                              )
                            : Container(
                                color: KidColors.color4.withAlpha(30),
                                child: const Center(
                                  child: Icon(
                                    Icons.photo_camera_back_rounded,
                                    size: 64,
                                    color: KidColors.black,
                                  ),
                                ),
                              ),
                      ),
                      if (result.markRegions.isNotEmpty)
                        Positioned.fill(
                          child: LayoutBuilder(
                            builder: (context, constraints) {
                              return Stack(
                                children: result.markRegions
                                    .map((region) => Positioned(
                                          left: constraints.maxWidth *
                                              region.left.clamp(0, 1),
                                          top: constraints.maxHeight *
                                              region.top.clamp(0, 1),
                                          width: constraints.maxWidth *
                                              region.width.clamp(0, 1),
                                          height: constraints.maxHeight *
                                              region.height.clamp(0, 1),
                                          child: Container(
                                            alignment: Alignment.topLeft,
                                            decoration: BoxDecoration(
                                              border: Border.all(
                                                color: region.isCorrect
                                                    ? KidColors.color3
                                                    : KidColors.color5,
                                                width: 3,
                                              ),
                                              borderRadius:
                                                  BorderRadius.circular(16),
                                              color: (region.isCorrect
                                                      ? KidColors.color3
                                                      : KidColors.color5)
                                                  .withAlpha(26),
                                            ),
                                            child: Container(
                                              margin: const EdgeInsets.all(6),
                                              padding:
                                                  const EdgeInsets.symmetric(
                                                      horizontal: 8,
                                                      vertical: 4),
                                              decoration: BoxDecoration(
                                                color: region.isCorrect
                                                    ? KidColors.color3
                                                    : KidColors.color5,
                                                borderRadius:
                                                    BorderRadius.circular(999),
                                              ),
                                              child: Text(
                                                region.markerLabel?.trim().isNotEmpty ==
                                                        true
                                                    ? region.markerLabel!
                                                    : (region.isCorrect
                                                        ? '✅'
                                                        : '❌'),
                                                style: const TextStyle(
                                                  color: Colors.white,
                                                  fontWeight: FontWeight.w900,
                                                ),
                                              ),
                                            ),
                                          ),
                                        ))
                                    .toList(),
                              );
                            },
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ),
            if (result.incorrectCount > 0) ...[
              const SizedBox(height: 16),
              Text(
                '需要订正的单词：${result.incorrectCount} 个',
                style: const TextStyle(fontWeight: FontWeight.w900),
              ),
            ],
          ],
        ),
      ),
    ),
  );
}

class _WordPlaybackPanel extends StatelessWidget {
  const _WordPlaybackPanel(
      {required this.state,
      required this.onLanguageChanged,
      required this.onModeChanged,
      required this.onSyncBackend,
      required this.onPlayCurrent,
      required this.onReplayCurrent,
      required this.onNextWord,
      required this.onPreviousWord,
      required this.onPeekChanged,
      required this.onSubmitPhoto});
  final WordPlaybackState state;
  final ValueChanged<WordPlaybackLanguage> onLanguageChanged;
  final ValueChanged<WordPlaybackMode> onModeChanged;
  final ValueChanged<bool> onPeekChanged;
  final VoidCallback onSyncBackend,
      onReplayCurrent,
      onNextWord,
      onPreviousWord,
      onSubmitPhoto;
  final Future<void> Function() onPlayCurrent;
  String get _instructionText {
    if (state.waitingForParentWordList) return '等家长补充词单后再来默写';
    if (!state.hasWords) return '请同步词单';
    if (state.session?.hasGradingResult == true) {
      return '看看这次批改结果，把需要订正的单词认真改好。';
    }
    if (state.session?.isGradingPending == true) {
      return '照片已经交给云端，先看看交卷进度，等批改结果回来。';
    }
    if (state.isPeeking) return '预览模式';

    if (state.language == WordPlaybackLanguage.english) {
      return state.mode == WordPlaybackMode.word
          ? '听到单词，请默写英文'
          : '听到释义，请默写英文单词';
    } else {
      return state.mode == WordPlaybackMode.word
          ? '听到词语，请默写中文'
          : '听到释义，请默写中文词语';
    }
  }

  String get _submitLabel {
    if (state.isBusy) return '上传中';
    if (state.session?.isGradingPending == true) return '后台批改中';
    if (state.session?.hasGradingResult == true) return '重新拍照交卷';
    return '📸 拍照交卷';
  }

  String get _syncLabel {
    if (state.isBusy) return '中...';
    if (state.waitingForParentWordList) return '重新同步';
    return '同步云端';
  }

  String get _playLabel {
    if (state.waitingForParentWordList) return '等待词单';
    if (state.isSpeaking) return '播报中';
    return '开始播报';
  }

  @override
  Widget build(BuildContext context) {
    return KidCard(
        borderColor: KidColors.black,
        padding: const EdgeInsets.all(32),
        child: Column(children: [
          Row(children: [
            const Text('听写舞台',
                style: TextStyle(fontSize: 26, fontWeight: FontWeight.w900)),
            const Spacer(),
            GestureDetector(
                onLongPressStart: (_) => onPeekChanged(true),
                onLongPressEnd: (_) => onPeekChanged(false),
                child: Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                        color: state.isPeeking
                            ? KidColors.color5
                            : KidColors.white,
                        shape: BoxShape.circle,
                        border: Border.all(color: KidColors.black, width: 2)),
                    child: Icon(
                        state.isPeeking
                            ? Icons.visibility_rounded
                            : Icons.visibility_off_rounded,
                        color:
                            state.isPeeking ? Colors.white : KidColors.black))),
            const SizedBox(width: 12),
            SizedBox(
                width: 140,
                child: KidSmallBtn(
                    label: _syncLabel,
                    color: KidColors.color2,
                    onTap: state.isBusy ? null : onSyncBackend)),
          ]),
          const SizedBox(height: 24),
          Wrap(spacing: 8, runSpacing: 8, children: [
            _TabBtn(
                label: '英语',
                isSel: state.language == WordPlaybackLanguage.english,
                onTap: () => onLanguageChanged(WordPlaybackLanguage.english)),
            _TabBtn(
                label: '语文',
                isSel: state.language == WordPlaybackLanguage.chinese,
                onTap: () => onLanguageChanged(WordPlaybackLanguage.chinese)),
            const SizedBox(width: 8),
            _TabBtn(
                label: '听词',
                isSel: state.mode == WordPlaybackMode.word,
                onTap: () => onModeChanged(WordPlaybackMode.word)),
            _TabBtn(
                label: '听义',
                isSel: state.mode == WordPlaybackMode.meaning,
                onTap: () => onModeChanged(WordPlaybackMode.meaning))
          ]),
          const SizedBox(height: 32),
          Container(
              width: double.infinity,
              padding: const EdgeInsets.symmetric(vertical: 80, horizontal: 32),
              decoration: BoxDecoration(
                  color: KidColors.color4,
                  borderRadius: BorderRadius.circular(32),
                  border: Border.all(color: KidColors.black, width: 3)),
              child: Column(children: [
                Text(
                    state.waitingForParentWordList
                        ? '待补充'
                        : state.hasWords
                            ? (state.isPeeking
                                ? state.currentWord
                                : (state.session?.hasGradingResult == true
                                    ? '本次听写结果'
                                    : state.session?.isGradingPending == true
                                        ? '等待批改'
                                        : '挑战 #${state.currentDisplayIndex}'))
                            : '等待中',
                    textAlign: TextAlign.center,
                    style: const TextStyle(
                        fontSize: 64,
                        fontWeight: FontWeight.w900,
                        color: KidColors.black)),
                const SizedBox(height: 16),
                Text(_instructionText,
                    style: TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.w700,
                        color: KidColors.black.withAlpha(120))),
                const SizedBox(height: 40),
                LayoutBuilder(builder: (context, constraints) {
                  return Stack(children: [
                    Container(
                        height: 16,
                        decoration: BoxDecoration(
                            color: KidColors.black.withAlpha(30),
                            borderRadius: BorderRadius.circular(8))),
                    AnimatedContainer(
                        duration: const Duration(milliseconds: 400),
                        height: 16,
                        width: constraints.maxWidth * state.progress,
                        decoration: BoxDecoration(
                            color: KidColors.black,
                            borderRadius: BorderRadius.circular(8))),
                  ]);
                }),
                const SizedBox(height: 16),
                Text(
                    state.session?.hasGradingResult == true
                        ? '结果已返回'
                        : state.session?.isGradingPending == true
                            ? '交卷已完成'
                            : '进度 ${state.currentDisplayIndex} / ${state.totalWords}',
                    style: const TextStyle(
                        fontWeight: FontWeight.w900, fontSize: 18)),
              ])),
          if (state.noticeMessage != null || state.errorMessage != null) ...[
            const SizedBox(height: 24),
            Container(
                padding: const EdgeInsets.all(20),
                width: double.infinity,
                decoration: BoxDecoration(
                    color: state.errorMessage != null
                        ? KidColors.color5.withAlpha(40)
                        : KidColors.color3.withAlpha(40),
                    borderRadius: BorderRadius.circular(24),
                    border: Border.all(
                        color: state.errorMessage != null
                            ? KidColors.color5
                            : KidColors.color3,
                        width: 2)),
                child: Row(children: [
                  Icon(
                      state.errorMessage != null
                          ? Icons.error_outline_rounded
                          : Icons.tips_and_updates_rounded,
                      color: state.errorMessage != null
                          ? KidColors.color5
                          : KidColors.color3),
                  const SizedBox(width: 12),
                  Expanded(
                      child: Text(state.errorMessage ?? state.noticeMessage!,
                          style: TextStyle(
                              fontWeight: FontWeight.w800,
                              color: state.errorMessage != null
                                  ? KidColors.color5
                                  : KidColors.color3.withAlpha(200),
                              fontSize: 16)))
                ]))
          ],
          if (state.lastSubmission != null) ...[
            const SizedBox(height: 24),
            _LastSubmissionPreviewCard(
              snapshot: state.lastSubmission!,
              session: state.session,
            ),
          ],
          if (state.session?.gradingStatus.isNotEmpty == true &&
              state.session?.gradingStatus != 'idle') ...[
            const SizedBox(height: 24),
            _GradingJourneyCard(session: state.session!),
            const SizedBox(height: 16),
            _GradingStatusCard(session: state.session!),
          ],
          const SizedBox(height: 32),
          Row(children: [
            Expanded(
                child: KidSmallBtn(
                    label: '上一个',
                    color: KidColors.color1,
                    onTap: state.canPrevious &&
                            !state.isBusy &&
                            !(state.session?.isShowingResultView ?? false)
                        ? onPreviousWord
                        : null)),
            const SizedBox(width: 12),
            Expanded(
                flex: 2,
                child: KidSmallBtn(
                    label: _playLabel,
                    color: KidColors.color3,
                    onTap: state.hasWords &&
                            !state.isSpeaking &&
                            !state.isBusy &&
                            !(state.session?.isShowingResultView ?? false)
                        ? onPlayCurrent
                        : null)),
            const SizedBox(width: 12),
            Expanded(
                child: KidSmallBtn(
                    label: state.canNext ? '下一个' : '已播完',
                    color: KidColors.color1,
                    onTap: state.canNext &&
                            !state.isBusy &&
                            !(state.session?.isShowingResultView ?? false)
                        ? onNextWord
                        : null))
          ]),
          const SizedBox(height: 16),
          Row(children: [
            Expanded(
                child: KidSmallBtn(
                    label: '重播',
                    color: KidColors.color5,
                    onTap: state.hasWords &&
                            !state.isBusy &&
                            !(state.session?.isShowingResultView ?? false)
                        ? onReplayCurrent
                        : null)),
            const SizedBox(width: 12),
            Expanded(
                child: KidSmallBtn(
                    label: _submitLabel,
                    color: KidColors.color5,
                    onTap: state.hasWords &&
                            !state.isBusy &&
                            !(state.session?.isGradingPending ?? false)
                        ? onSubmitPhoto
                        : null)),
          ]),
        ]));
  }
}

class _LastSubmissionPreviewCard extends StatelessWidget {
  const _LastSubmissionPreviewCard({
    required this.snapshot,
    required this.session,
  });

  final DictationSubmissionSnapshot snapshot;
  final DictationSession? session;

  @override
  Widget build(BuildContext context) {
    final stageMeta = describeDictationStage(session);
    final photoToken = _shortTraceToken(session?.debugContext?.photoSha1);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: KidColors.color4.withAlpha(18),
        borderRadius: BorderRadius.circular(24),
        border: Border.all(color: KidColors.color4, width: 2),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            session?.hasGradingResult == true ? '最近一次交卷照片' : '最近一次交卷',
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
          ),
          const SizedBox(height: 8),
          Text(
            session?.hasGradingResult == true
                ? '批改结果已经回来了，先看看这次照片和标注。'
                : '先看看照片是否清楚，再等 AI 给结果。',
            style: TextStyle(
              fontSize: 15,
              fontWeight: FontWeight.w800,
              color: KidColors.black.withAlpha(170),
            ),
          ),
          const SizedBox(height: 16),
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Stack(
                children: [
                  Container(
                    width: 120,
                    height: 120,
                    decoration: BoxDecoration(
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(22),
                      border: Border.all(color: KidColors.black, width: 2),
                    ),
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(20),
                      child: snapshot.previewBytes != null
                          ? Image.memory(
                              snapshot.previewBytes!,
                              fit: BoxFit.cover,
                              gaplessPlayback: true,
                            )
                          : const Icon(
                              Icons.photo_camera_back_rounded,
                              size: 44,
                              color: KidColors.black,
                            ),
                    ),
                  ),
                  if (session?.gradingResult?.hasMarkRegions == true)
                    Positioned(
                      left: 8,
                      top: 8,
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 8, vertical: 4),
                        decoration: BoxDecoration(
                          color: KidColors.color5,
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: const Text(
                          '含错词标注',
                          style: TextStyle(
                            color: Colors.white,
                            fontWeight: FontWeight.w900,
                            fontSize: 12,
                          ),
                        ),
                      ),
                    ),
                  if (session?.gradingResult?.hasAnnotatedPhoto == true ||
                      session?.gradingResult?.hasMarkRegions == true)
                    Positioned(
                      right: 8,
                      bottom: 8,
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 8, vertical: 4),
                        decoration: BoxDecoration(
                          color: KidColors.black.withAlpha(180),
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: const Text(
                          '点结果卡可放大',
                          style: TextStyle(
                            color: Colors.white,
                            fontWeight: FontWeight.w900,
                            fontSize: 12,
                          ),
                        ),
                      ),
                    ),
                ],
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: [
                        _MiniTraceChip(
                            label:
                                _formatSubmissionClock(snapshot.submittedAt)),
                        _MiniTraceChip(
                            label: _formatSubmissionBytes(snapshot.byteCount)),
                        _MiniTraceChip(label: stageMeta.label),
                        if (photoToken.isNotEmpty)
                          _MiniTraceChip(label: '图片 $photoToken'),
                      ],
                    ),
                    const SizedBox(height: 14),
                    Text(
                      stageMeta.hint,
                      style: TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w700,
                        color: KidColors.black.withAlpha(165),
                      ),
                    ),
                    const SizedBox(height: 12),
                    Text(
                      '如果模糊、歪斜或没拍全，现在就重拍一张。',
                      style: TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w800,
                        color: KidColors.color5.withAlpha(210),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _GradingJourneyCard extends StatelessWidget {
  const _GradingJourneyCard({required this.session});

  final DictationSession session;

  @override
  Widget build(BuildContext context) {
    final stageMeta = describeDictationStage(session);
    final tracePills = _buildTracePills(session);
    final steps = _buildJourneySteps(session);

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: KidColors.color2.withAlpha(20),
        borderRadius: BorderRadius.circular(24),
        border: Border.all(color: KidColors.color2, width: 2),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            '交卷进度',
            style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
          ),
          const SizedBox(height: 8),
          Text(
            '${stageMeta.label} · ${stageMeta.hint}',
            style: TextStyle(
              fontSize: 15,
              fontWeight: FontWeight.w800,
              color: KidColors.color1.withAlpha(210),
            ),
          ),
          const SizedBox(height: 16),
          ...steps.map((step) => Padding(
                padding: const EdgeInsets.only(bottom: 10),
                child: _JourneyStepCard(step: step),
              )),
          if (tracePills.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(
              '同步线索',
              style: TextStyle(
                fontSize: 13,
                fontWeight: FontWeight.w900,
                color: KidColors.black.withAlpha(160),
              ),
            ),
            const SizedBox(height: 10),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: tracePills
                  .map((item) => _MiniTraceChip(label: item))
                  .toList(),
            ),
          ],
        ],
      ),
    );
  }
}

class _JourneyStepCard extends StatelessWidget {
  const _JourneyStepCard({required this.step});

  final _JourneyStepData step;

  @override
  Widget build(BuildContext context) {
    Color accent;
    IconData icon;
    switch (step.state) {
      case _JourneyStepVisualState.complete:
        accent = KidColors.color3;
        icon = Icons.check_circle_rounded;
        break;
      case _JourneyStepVisualState.active:
        accent = KidColors.color4;
        icon = Icons.autorenew_rounded;
        break;
      case _JourneyStepVisualState.failed:
        accent = KidColors.color5;
        icon = Icons.error_outline_rounded;
        break;
      case _JourneyStepVisualState.pending:
        accent = KidColors.black.withAlpha(120);
        icon = Icons.radio_button_unchecked_rounded;
        break;
    }

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: accent, width: 2),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 38,
            height: 38,
            decoration: BoxDecoration(
              color: accent.withAlpha(35),
              shape: BoxShape.circle,
            ),
            child: Icon(icon, color: accent),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  step.title,
                  style: const TextStyle(
                      fontSize: 16, fontWeight: FontWeight.w900),
                ),
                const SizedBox(height: 4),
                Text(
                  step.caption,
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w700,
                    color: KidColors.black.withAlpha(170),
                  ),
                ),
              ],
            ),
          ),
          if (step.timeLabel != null)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
              decoration: BoxDecoration(
                color: accent.withAlpha(28),
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                step.timeLabel!,
                style: TextStyle(
                  fontWeight: FontWeight.w900,
                  color: accent,
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _MiniTraceChip extends StatelessWidget {
  const _MiniTraceChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: KidColors.black, width: 1.5),
      ),
      child: Text(
        label,
        style: const TextStyle(fontWeight: FontWeight.w900, fontSize: 13),
      ),
    );
  }
}

class _GradingStatusCard extends StatelessWidget {
  const _GradingStatusCard({required this.session});

  final DictationSession session;

  @override
  Widget build(BuildContext context) {
    final result = session.gradingResult;
    final stageMeta = describeDictationStage(session);
    final incorrectItems = result?.gradedItems
            .where((item) => !item.isCorrect || item.needsCorrection)
            .toList() ??
        const <DictationGradedItem>[];

    Color borderColor;
    Color backgroundColor;
    IconData icon;
    String title;
    String subtitle;

    switch (session.gradingStatus) {
      case 'pending':
      case 'processing':
        borderColor = KidColors.color2;
        backgroundColor = KidColors.color2.withAlpha(35);
        icon = Icons.hourglass_top_rounded;
        title = '交卷进行中 · ${stageMeta.label}';
        subtitle = stageMeta.hint;
        break;
      case 'failed':
        borderColor = KidColors.color5;
        backgroundColor = KidColors.color5.withAlpha(35);
        icon = Icons.error_outline_rounded;
        title = '这次受阻了';
        subtitle = session.gradingError?.isNotEmpty == true
            ? session.gradingError!
            : stageMeta.hint;
        break;
      case 'completed':
        borderColor = result?.incorrectCount == 0
            ? KidColors.color3
            : KidColors.color5;
        backgroundColor = borderColor.withAlpha(35);
        icon = result?.incorrectCount == 0
            ? Icons.emoji_events_rounded
            : Icons.fact_check_rounded;
        title = result == null
            ? '本次交卷已批改'
            : (result.incorrectCount == 0
                ? '本次听写全对啦'
                : '本次听写结果');
        subtitle = result?.aiFeedback.isNotEmpty == true
            ? result!.aiFeedback
            : stageMeta.hint;
        break;
      default:
        borderColor = KidColors.black;
        backgroundColor = KidColors.white;
        icon = Icons.info_outline_rounded;
        title = '批改状态';
        subtitle = '等待后台状态同步。';
    }

    final canPreviewAnnotated =
        result?.hasAnnotatedPhoto == true || result?.hasMarkRegions == true;

    return GestureDetector(
      onTap: canPreviewAnnotated ? () => _showAnnotatedResultPreview(context, result!) : null,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: backgroundColor,
          borderRadius: BorderRadius.circular(24),
          border: Border.all(color: borderColor, width: 2),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, color: borderColor),
                const SizedBox(width: 12),
                Expanded(
                    child: Text(title,
                        style: const TextStyle(
                            fontSize: 18, fontWeight: FontWeight.w900))),
                if (canPreviewAnnotated)
                  Container(
                    padding:
                        const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                    decoration: BoxDecoration(
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(999),
                      border: Border.all(color: borderColor, width: 1.5),
                    ),
                    child: Text(
                      '放大查看',
                      style: TextStyle(
                        fontWeight: FontWeight.w900,
                        color: borderColor,
                      ),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 8),
            Text(subtitle,
                style:
                    const TextStyle(fontSize: 15, fontWeight: FontWeight.w700)),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                _MiniTraceChip(label: '会话 ${session.sessionId}'),
                if (session.gradingRequestedAt?.isNotEmpty == true)
                  _MiniTraceChip(
                      label:
                          '提交 ${_formatJourneyTime(session.gradingRequestedAt)}'),
                if (session.gradingCompletedAt?.isNotEmpty == true)
                  _MiniTraceChip(
                      label:
                          '返回 ${_formatJourneyTime(session.gradingCompletedAt)}'),
              ],
            ),
            if (result != null) ...[
              const SizedBox(height: 16),
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(20),
                  border: Border.all(color: borderColor, width: 1.5),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      result.incorrectCount <= 0 ? '满分表现' : '结果小结',
                      style: const TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.w900,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Text(
                      _formatScoreSummary(result),
                      style: const TextStyle(
                        fontSize: 15,
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                    if (result.incorrectCount <= 0) ...[
                      const SizedBox(height: 8),
                      Text(
                        '太棒了，所有单词都写对了，继续保持这样的专注。',
                        style: TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w800,
                          color: KidColors.color3.withAlpha(220),
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ],
            if (incorrectItems.isNotEmpty) ...[
              const SizedBox(height: 16),
              const Text(
                '这次需要订正的单词',
                style: TextStyle(fontSize: 16, fontWeight: FontWeight.w900),
              ),
              const SizedBox(height: 10),
              ...incorrectItems.map(
                (item) => Container(
                  width: double.infinity,
                  margin: const EdgeInsets.only(bottom: 10),
                  padding: const EdgeInsets.all(14),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(18),
                    border: Border.all(color: KidColors.color5, width: 1.5),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Container(
                            width: 28,
                            height: 28,
                            alignment: Alignment.center,
                            decoration: const BoxDecoration(
                              color: KidColors.color5,
                              shape: BoxShape.circle,
                            ),
                            child: const Text(
                              '❌',
                              style: TextStyle(fontSize: 14),
                            ),
                          ),
                          const SizedBox(width: 10),
                          Expanded(
                            child: Text(
                              '第 ${item.index} 词 · ${item.expected}',
                              style: const TextStyle(
                                fontWeight: FontWeight.w900,
                                fontSize: 15,
                              ),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 10),
                      Text(
                        '正确写法：${item.expected}',
                        style: const TextStyle(fontWeight: FontWeight.w800),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        '你这次写成：${item.actual.trim().isEmpty ? '未识别清楚' : item.actual}',
                        style: TextStyle(
                          fontWeight: FontWeight.w800,
                          color: KidColors.color5.withAlpha(220),
                        ),
                      ),
                      if ((item.meaning?.trim().isNotEmpty ?? false)) ...[
                        const SizedBox(height: 4),
                        Text(
                          '词义：${item.meaning!}',
                          style: TextStyle(
                            fontWeight: FontWeight.w700,
                            color: KidColors.black.withAlpha(170),
                          ),
                        ),
                      ],
                      const SizedBox(height: 8),
                      Text(
                        _formatWordCorrectionComment(item),
                        style: TextStyle(
                          fontWeight: FontWeight.w700,
                          color: KidColors.black.withAlpha(180),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
            if (result != null &&
                incorrectItems.isEmpty &&
                canPreviewAnnotated) ...[
              const SizedBox(height: 12),
              Text(
                '点开可以放大查看批改标注图。',
                style: TextStyle(
                  fontWeight: FontWeight.w800,
                  color: borderColor.withAlpha(220),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _HomeModeSwitcher extends StatelessWidget {
  const _HomeModeSwitcher({required this.selectedTab, required this.onChanged});
  final _PadHomeTab selectedTab;
  final ValueChanged<_PadHomeTab> onChanged;
  @override
  Widget build(BuildContext context) => Row(children: [
        Expanded(
            child: _TabBtn(
                label: '今日挑战',
                isSel: selectedTab == _PadHomeTab.tasks,
                onTap: () => onChanged(_PadHomeTab.tasks))),
        const SizedBox(width: 12),
        Expanded(
            child: _TabBtn(
                label: '听写练词',
                isSel: selectedTab == _PadHomeTab.words,
                onTap: () => onChanged(_PadHomeTab.words)))
      ]);
}

class _TabBtn extends StatelessWidget {
  const _TabBtn(
      {required this.label, required this.isSel, required this.onTap});
  final String label;
  final bool isSel;
  final VoidCallback onTap;
  @override
  Widget build(BuildContext context) => GestureDetector(
      onTap: onTap,
      child: Container(
          padding: const EdgeInsets.symmetric(vertical: 12),
          alignment: Alignment.center,
          decoration: BoxDecoration(
              color: isSel ? KidColors.black : KidColors.white,
              borderRadius: BorderRadius.circular(16),
              border: Border.all(color: KidColors.black, width: 2)),
          child: Text(label,
              style: TextStyle(
                  color: isSel ? KidColors.white : KidColors.black,
                  fontWeight: FontWeight.w900))));
}

class _EmptyBoard extends StatelessWidget {
  const _EmptyBoard({required this.title, required this.description});
  final String title, description;
  @override
  Widget build(BuildContext context) => Center(
          child: Column(children: [
        const Icon(Icons.coffee_rounded, size: 64, color: KidColors.color2),
        const SizedBox(height: 16),
        Text(title,
            style: const TextStyle(fontSize: 22, fontWeight: FontWeight.w900)),
        Text(description)
      ]));
}

class _BannerCard extends StatelessWidget {
  const _BannerCard({required this.message});
  final String message;
  @override
  Widget build(BuildContext context) => Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
          color: KidColors.color5, borderRadius: BorderRadius.circular(12)),
      child: Row(children: [
        const Icon(Icons.error_outline_rounded, color: Colors.white),
        const SizedBox(width: 12),
        Expanded(
            child: Text(message,
                style: const TextStyle(
                    color: Colors.white, fontWeight: FontWeight.w700)))
      ]));
}
