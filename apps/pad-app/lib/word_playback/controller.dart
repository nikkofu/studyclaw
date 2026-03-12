import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/repository.dart';
import 'package:pad_app/word_playback/models.dart';
import 'package:pad_app/word_playback/speaker_contract.dart';

const Object _missing = Object();

class WordPlaybackState {
  const WordPlaybackState({
    required this.language,
    this.mode = WordPlaybackMode.word,
    this.words = const <String>[],
    this.currentIndex = 0,
    this.isSpeaking = false,
    this.isBusy = false,
    this.isPeeking = false,
    this.errorMessage,
    this.noticeMessage,
    this.wordList,
    this.session,
    this.lastSubmission,
  });

  factory WordPlaybackState.initial({
    WordPlaybackLanguage language = WordPlaybackLanguage.english,
  }) {
    return WordPlaybackState(language: language);
  }

  final WordPlaybackLanguage language;
  final WordPlaybackMode mode;
  final List<String> words;
  final int currentIndex;
  final bool isSpeaking;
  final bool isBusy;
  final bool isPeeking;
  final String? errorMessage;
  final String? noticeMessage;
  final WordList? wordList;
  final DictationSession? session;
  final DictationSubmissionSnapshot? lastSubmission;

  bool get hasWords => words.isNotEmpty || session?.currentItem != null;

  String get currentWord {
    if (session?.currentItem != null) {
      return session!.currentItem!.text;
    }
    if (words.isEmpty) return '';
    return words[currentIndex];
  }

  String get currentSpeakingContent {
    if (session?.currentItem != null) {
      final item = session!.currentItem!;
      return mode == WordPlaybackMode.word
          ? item.text
          : (item.meaning ?? item.text);
    }
    return currentWord;
  }

  int get currentDisplayIndex {
    if (session != null) {
      return (session!.currentIndex + 1).clamp(0, totalWords);
    }
    if (words.isEmpty) return 0;
    return (currentIndex + 1).clamp(0, totalWords);
  }

  int get totalWords {
    if (session != null) return session!.totalItems;
    return words.length;
  }

  double get progress {
    if (totalWords == 0) return 0;
    return (currentDisplayIndex / totalWords).clamp(0.0, 1.0);
  }

  bool get canNext {
    if (session != null) return !session!.isCompleted;
    return words.isNotEmpty && currentIndex < words.length - 1;
  }

  bool get canPrevious {
    if (session != null) return session!.currentIndex > 1;
    return currentIndex > 0;
  }

  WordPlaybackState copyWith({
    WordPlaybackLanguage? language,
    WordPlaybackMode? mode,
    Object? words = _missing,
    int? currentIndex,
    bool? isSpeaking,
    bool? isBusy,
    bool? isPeeking,
    Object? errorMessage = _missing,
    Object? noticeMessage = _missing,
    Object? wordList = _missing,
    Object? session = _missing,
    Object? lastSubmission = _missing,
  }) {
    return WordPlaybackState(
      language: language ?? this.language,
      mode: mode ?? this.mode,
      words: words == _missing ? this.words : words as List<String>,
      currentIndex: currentIndex ?? this.currentIndex,
      isSpeaking: isSpeaking ?? this.isSpeaking,
      isBusy: isBusy ?? this.isBusy,
      isPeeking: isPeeking ?? this.isPeeking,
      errorMessage: errorMessage == _missing
          ? this.errorMessage
          : errorMessage as String?,
      noticeMessage: noticeMessage == _missing
          ? this.noticeMessage
          : noticeMessage as String?,
      wordList: wordList == _missing ? this.wordList : wordList as WordList?,
      session:
          session == _missing ? this.session : session as DictationSession?,
      lastSubmission: lastSubmission == _missing
          ? this.lastSubmission
          : lastSubmission as DictationSubmissionSnapshot?,
    );
  }
}

class DictationSubmissionSnapshot {
  const DictationSubmissionSnapshot({
    required this.submittedAt,
    required this.byteCount,
    this.previewBytes,
  });

  final DateTime submittedAt;
  final int byteCount;
  final Uint8List? previewBytes;
}

class WordPlaybackController extends ChangeNotifier {
  WordPlaybackController({
    required WordSpeaker speaker,
    required TaskBoardRepository repository,
    WordPlaybackLanguage initialLanguage = WordPlaybackLanguage.english,
  })  : _speaker = speaker,
        _repository = repository,
        _state = WordPlaybackState.initial(language: initialLanguage);

  final WordSpeaker _speaker;
  final TaskBoardRepository _repository;

  WordPlaybackState _state;
  bool _disposed = false;
  int _gradingPollToken = 0;

  WordPlaybackState get state => _state;
  bool get supportsPlayback => _speaker.supportsPlayback;

  void setLanguage(WordPlaybackLanguage language) {
    if (_state.language == language) return;
    _state = _state.copyWith(
        language: language,
        errorMessage: null,
        noticeMessage: '已切换到${language.label}播放模式');
    notifyListeners();
  }

  void setMode(WordPlaybackMode mode) {
    if (_state.mode == mode) return;
    _state = _state.copyWith(mode: mode, noticeMessage: '已切换到${mode.label}模式');
    notifyListeners();
  }

  void setPeeking(bool peeking) {
    if (_state.isPeeking == peeking) return;
    _state = _state.copyWith(isPeeking: peeking);
    notifyListeners();
  }

  Future<void> syncWordList(TaskBoardRequest request) async {
    _state = _state.copyWith(
        isBusy: true, errorMessage: null, noticeMessage: '正在同步词单...');
    notifyListeners();

    try {
      final wordList = await _repository.fetchWordList(request);
      final words = wordList.items.map((item) => item.text).toList();
      _state = _state.copyWith(
        isBusy: false,
        wordList: wordList,
        words: words,
        currentIndex: 0,
        session: null,
        lastSubmission: null,
        language: wordList.language,
        noticeMessage: '已同步：${wordList.title} (${wordList.totalItems}个词)',
      );
      notifyListeners();
    } catch (error) {
      _state = _state.copyWith(isBusy: false, errorMessage: '同步词单失败：$error');
    }
    notifyListeners();
  }

  Future<void> startDictation(TaskBoardRequest request) async {
    _state = _state.copyWith(
        isBusy: true, errorMessage: null, noticeMessage: '正在开启听写会话...');
    notifyListeners();

    try {
      final session = await _repository.startDictationSession(request);
      _state = _state.copyWith(
          isBusy: false,
          session: session,
          lastSubmission: null,
          noticeMessage: '听写会话已开启，准备开始播放。');
      if (session.currentItem != null) await playCurrent();
    } catch (error) {
      _state = _state.copyWith(isBusy: false, errorMessage: '开启听写失败：$error');
    }
    notifyListeners();
  }

  Future<void> playCurrent() async {
    final word = _state.currentSpeakingContent;
    if (word.isEmpty) {
      _state = _state.copyWith(errorMessage: '当前没有待播放的单词。');
      notifyListeners();
      return;
    }

    _state = _state.copyWith(
        isSpeaking: true,
        errorMessage: null,
        noticeMessage:
            '正在播报：${_state.mode.label} (${_state.currentDisplayIndex}/${_state.totalWords})');
    notifyListeners();

    try {
      await _speaker.speak(word, language: _state.language);
    } catch (error) {
      _state = _state.copyWith(isSpeaking: false, errorMessage: '播放失败：$error');
      notifyListeners();
      return;
    }

    _state = _state.copyWith(isSpeaking: false);
    notifyListeners();
  }

  Future<void> replayCurrent(String apiBaseUrl) async {
    if (_state.session != null) {
      _state = _state.copyWith(isBusy: true);
      notifyListeners();
      try {
        final session = await _repository.replayDictationSession(
            _state.session!.sessionId, apiBaseUrl);
        _state = _state.copyWith(isBusy: false, session: session);
      } catch (error) {
        _state = _state.copyWith(isBusy: false, errorMessage: '重播请求失败：$error');
      }
      notifyListeners();
    }
    return playCurrent();
  }

  Future<void> nextWord(String apiBaseUrl) async {
    if (_state.session != null) {
      _state = _state.copyWith(isBusy: true);
      notifyListeners();
      try {
        final session = await _repository.nextDictationSession(
            _state.session!.sessionId, apiBaseUrl);
        _state = _state.copyWith(
            isBusy: false,
            session: session,
            isSpeaking: false,
            noticeMessage: session.isCompleted ? '这组单词已经播报完啦！' : '已切换到下一词');
        notifyListeners();
        if (!session.isCompleted) await playCurrent();
        return;
      } catch (error) {
        _state = _state.copyWith(isBusy: false, errorMessage: '切换下一词失败：$error');
        notifyListeners();
        return;
      }
    }

    if (state.canNext) {
      _state = _state.copyWith(currentIndex: _state.currentIndex + 1);
      notifyListeners();
      await playCurrent();
    }
  }

  Future<void> previousWord(String apiBaseUrl) async {
    if (_state.session != null) {
      _state = _state.copyWith(isBusy: true);
      notifyListeners();
      try {
        final session = await _repository.previousDictationSession(
            _state.session!.sessionId, apiBaseUrl);
        _state = _state.copyWith(
            isBusy: false,
            session: session,
            isSpeaking: false,
            noticeMessage: '已返回上一词');
        notifyListeners();
        await playCurrent();
        return;
      } catch (error) {
        _state = _state.copyWith(isBusy: false, errorMessage: '切换上一词失败：$error');
        notifyListeners();
        return;
      }
    }

    if (state.canPrevious) {
      _state = _state.copyWith(currentIndex: _state.currentIndex - 1);
      notifyListeners();
      await playCurrent();
    }
  }

  Future<void> submitPhotoForGrading(
    String apiBaseUrl,
    String base64Image, {
    Uint8List? previewBytes,
    DateTime? submittedAt,
  }) async {
    if (_state.session == null) {
      _state = _state.copyWith(errorMessage: '当前还没有听写会话，不能交卷。');
      notifyListeners();
      return;
    }
    final sessionId = _state.session!.sessionId;
    final pollToken = ++_gradingPollToken;
    final submissionSnapshot = DictationSubmissionSnapshot(
      submittedAt: submittedAt ?? DateTime.now(),
      byteCount: previewBytes?.length ?? _estimateBase64Bytes(base64Image),
      previewBytes: previewBytes,
    );

    _state = _state.copyWith(
      isBusy: true,
      errorMessage: null,
      noticeMessage: '正在上传照片，准备交给后台批改...',
      lastSubmission: submissionSnapshot,
    );
    notifyListeners();

    try {
      final queuedSession = await _repository.gradeDictationSession(
        sessionId: sessionId,
        apiBaseUrl: apiBaseUrl,
        photoBase64: base64Image,
        language: _state.language.name,
        mode: _state.mode.name,
      );

      _state = _state.copyWith(
        isBusy: false,
        session: queuedSession,
        lastSubmission: submissionSnapshot,
        noticeMessage: _buildPendingNotice(queuedSession),
      );
      notifyListeners();
      unawaited(_pollGradingStatus(apiBaseUrl, sessionId, pollToken));
    } catch (error) {
      _state = _state.copyWith(isBusy: false, errorMessage: '批改失败：$error');
      notifyListeners();
    }
  }

  @override
  void dispose() {
    _disposed = true;
    _gradingPollToken++;
    _speaker.stop();
    super.dispose();
  }

  Future<void> _pollGradingStatus(
      String apiBaseUrl, String sessionId, int pollToken) async {
    const maxAttempts = 40;
    for (var attempt = 0; attempt < maxAttempts; attempt++) {
      await Future<void>.delayed(Duration(seconds: attempt == 0 ? 2 : 3));
      if (_disposed || pollToken != _gradingPollToken) {
        return;
      }

      try {
        final session =
            await _repository.getDictationSession(sessionId, apiBaseUrl);
        if (_disposed || pollToken != _gradingPollToken) {
          return;
        }

        if (session.isGradingPending) {
          _state = _state.copyWith(
            session: session,
            errorMessage: null,
            noticeMessage: _buildPendingNotice(session),
          );
          notifyListeners();
          continue;
        }

        if (session.gradingStatus == 'failed') {
          _state = _state.copyWith(
            session: session,
            errorMessage: _buildFailureMessage(session),
            noticeMessage: '这次没有顺利完成，可以重新拍照再试一次。',
          );
          notifyListeners();
          return;
        }

        if (session.hasGradingResult) {
          final result = session.gradingResult!;
          _state = _state.copyWith(
            session: session,
            errorMessage: null,
            noticeMessage:
                'AI 批改完成！得分 ${result.score}，${result.incorrectCount} 处需要订正。',
          );
          notifyListeners();
          return;
        }

        _state = _state.copyWith(
          session: session,
          errorMessage: null,
          noticeMessage: '后台批改状态已刷新。',
        );
        notifyListeners();
        return;
      } catch (error) {
        if (attempt == maxAttempts - 1) {
          _state = _state.copyWith(
            errorMessage: '已提交到后台，但轮询结果失败：$error',
            noticeMessage: '可稍后重新进入页面查看批改结果。',
          );
          notifyListeners();
        }
      }
    }

    if (_disposed || pollToken != _gradingPollToken) {
      return;
    }

    _state = _state.copyWith(
      errorMessage: null,
      noticeMessage: '后台批改仍在继续，可稍后刷新查看结果。',
    );
    notifyListeners();
  }
}

int _estimateBase64Bytes(String value) {
  final normalized = value.trim();
  if (normalized.isEmpty) {
    return 0;
  }

  var padding = 0;
  if (normalized.endsWith('==')) {
    padding = 2;
  } else if (normalized.endsWith('=')) {
    padding = 1;
  }
  return ((normalized.length * 3) ~/ 4) - padding;
}

String _buildPendingNotice(DictationSession session) {
  final stage = describeDictationStage(session);
  return '交卷进度：${stage.label}。${stage.hint}';
}

String _buildFailureMessage(DictationSession session) {
  final explicitError = session.gradingError?.trim() ?? '';
  if (explicitError.isNotEmpty) {
    return explicitError;
  }
  return describeDictationStage(session).hint;
}
