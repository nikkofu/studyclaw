import 'package:flutter/foundation.dart';
import 'package:pad_app/task_board/models.dart';
import 'package:pad_app/task_board/repository.dart';
import 'package:pad_app/word_playback/models.dart';
import 'package:pad_app/word_playback/speaker_contract.dart';

const Object _missing = Object();

class WordPlaybackState {
  const WordPlaybackState({
    required this.language,
    this.words = const <String>[],
    this.currentIndex = 0,
    this.isSpeaking = false,
    this.isBusy = false,
    this.errorMessage,
    this.noticeMessage,
    this.wordList,
    this.session,
  });

  factory WordPlaybackState.initial({
    WordPlaybackLanguage language = WordPlaybackLanguage.english,
  }) {
    return WordPlaybackState(language: language);
  }

  final WordPlaybackLanguage language;
  final List<String> words;
  final int currentIndex;
  final bool isSpeaking;
  final bool isBusy;
  final String? errorMessage;
  final String? noticeMessage;
  final WordList? wordList;
  final DictationSession? session;

  bool get hasWords => words.isNotEmpty;

  String get currentWord {
    if (session?.currentItem != null) {
      return session!.currentItem!.text;
    }
    if (!hasWords) {
      return '';
    }
    return words[currentIndex];
  }

  int get currentDisplayIndex {
    if (session != null) {
      return session!.currentIndex;
    }
    if (!hasWords) {
      return 0;
    }
    return currentIndex + 1;
  }

  int get totalWords {
    if (session != null) {
      return session!.totalItems;
    }
    return words.length;
  }

  double get progress {
    final total = totalWords;
    if (total == 0) {
      return 0;
    }
    return currentDisplayIndex / total;
  }

  bool get canNext {
    if (session != null) {
      return !session!.isCompleted;
    }
    return hasWords && currentIndex < words.length - 1;
  }

  WordPlaybackState copyWith({
    WordPlaybackLanguage? language,
    Object? words = _missing,
    int? currentIndex,
    bool? isSpeaking,
    bool? isBusy,
    Object? errorMessage = _missing,
    Object? noticeMessage = _missing,
    Object? wordList = _missing,
    Object? session = _missing,
  }) {
    return WordPlaybackState(
      language: language ?? this.language,
      words: words == _missing ? this.words : words as List<String>,
      currentIndex: currentIndex ?? this.currentIndex,
      isSpeaking: isSpeaking ?? this.isSpeaking,
      isBusy: isBusy ?? this.isBusy,
      errorMessage: errorMessage == _missing
          ? this.errorMessage
          : errorMessage as String?,
      noticeMessage: noticeMessage == _missing
          ? this.noticeMessage
          : noticeMessage as String?,
      wordList: wordList == _missing ? this.wordList : wordList as WordList?,
      session: session == _missing ? this.session : session as DictationSession?,
    );
  }
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

  WordPlaybackState get state => _state;
  bool get supportsPlayback => _speaker.supportsPlayback;

  void setLanguage(WordPlaybackLanguage language) {
    if (_state.language == language) {
      return;
    }
    _state = _state.copyWith(
      language: language,
      errorMessage: null,
      noticeMessage: '已切换到${language.label}播放模式',
    );
    notifyListeners();
  }

  Future<void> syncWordList(TaskBoardRequest request) async {
    _state = _state.copyWith(isBusy: true, errorMessage: null, noticeMessage: '正在同步词单...');
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
        language: wordList.language,
        noticeMessage: '已同步：${wordList.title} (${wordList.totalItems}个词)',
      );
    } catch (error) {
      _state = _state.copyWith(
        isBusy: false,
        errorMessage: '同步词单失败：$error',
      );
    }
    notifyListeners();
  }

  Future<void> startDictation(TaskBoardRequest request) async {
    _state = _state.copyWith(isBusy: true, errorMessage: null, noticeMessage: '正在开启听写会话...');
    notifyListeners();

    try {
      final session = await _repository.startDictationSession(request);
      _state = _state.copyWith(
        isBusy: false,
        session: session,
        noticeMessage: '听写会话已开启，准备开始播放。',
      );
      if (session.currentItem != null) {
        await playCurrent();
      }
    } catch (error) {
      _state = _state.copyWith(
        isBusy: false,
        errorMessage: '开启听写失败：$error',
      );
    }
    notifyListeners();
  }

  void loadWordsFromText(String rawText) {
    final words = parseWordEntries(rawText);
    if (words.isEmpty) {
      _state = _state.copyWith(
        words: const <String>[],
        currentIndex: 0,
        isSpeaking: false,
        errorMessage: '请先输入至少一个单词或词语。',
        noticeMessage: null,
        wordList: null,
        session: null,
      );
      notifyListeners();
      return;
    }

    _state = _state.copyWith(
      words: words,
      currentIndex: 0,
      isSpeaking: false,
      errorMessage: null,
      noticeMessage: '已载入 ${words.length} 个${_state.language.label}词条',
      wordList: null,
      session: null,
    );
    notifyListeners();
  }

  Future<void> playCurrent() async {
    final word = _state.currentWord;
    if (word.isEmpty) {
      _state = _state.copyWith(
        errorMessage: '当前没有待播放的单词。',
        noticeMessage: null,
      );
      notifyListeners();
      return;
    }

    if (!_speaker.supportsPlayback) {
      _state = _state.copyWith(
        isSpeaking: false,
        errorMessage: null,
        noticeMessage: '当前设备暂不支持自动朗读，可继续看词卡跟读。',
      );
      notifyListeners();
      return;
    }

    _state = _state.copyWith(
      isSpeaking: true,
      errorMessage: null,
      noticeMessage:
          '正在播放第 ${_state.currentDisplayIndex}/${_state.totalWords} 个词',
    );
    notifyListeners();

    try {
      await _speaker.speak(
        word,
        language: _state.language,
      );
    } catch (error) {
      _state = _state.copyWith(
        isSpeaking: false,
        errorMessage: '播放失败：$error',
        noticeMessage: null,
      );
      notifyListeners();
      return;
    }

    _state = _state.copyWith(
      isSpeaking: false,
      errorMessage: null,
      noticeMessage: '已播放“$word”',
    );
    notifyListeners();
  }

  Future<void> replayCurrent(String apiBaseUrl) async {
    if (_state.session != null) {
      _state = _state.copyWith(isBusy: true);
      notifyListeners();
      try {
        final session = await _repository.replayDictationSession(_state.session!.sessionId, apiBaseUrl);
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
        final session = await _repository.nextDictationSession(_state.session!.sessionId, apiBaseUrl);
        _state = _state.copyWith(
          isBusy: false,
          session: session,
          isSpeaking: false,
          errorMessage: null,
          noticeMessage: session.isCompleted 
            ? '这组单词已经播放完成啦' 
            : '已切换到第 ${session.currentIndex}/${session.totalItems} 个词',
        );
        notifyListeners();
        if (!session.isCompleted) {
          await playCurrent();
        }
        return;
      } catch (error) {
        _state = _state.copyWith(isBusy: false, errorMessage: '切换下一词失败：$error');
        notifyListeners();
        return;
      }
    }

    if (!_state.hasWords) {
      _state = _state.copyWith(
        errorMessage: '请先载入单词清单。',
        noticeMessage: null,
      );
      notifyListeners();
      return;
    }

    if (!_state.canNext) {
      _state = _state.copyWith(
        isSpeaking: false,
        errorMessage: null,
        noticeMessage: '这组单词已经播放完成啦',
      );
      notifyListeners();
      return;
    }

    _state = _state.copyWith(
      currentIndex: _state.currentIndex + 1,
      isSpeaking: false,
      errorMessage: null,
      noticeMessage:
          '已切换到第 ${_state.currentDisplayIndex + 1}/${_state.words.length} 个词',
    );
    notifyListeners();

    await playCurrent();
  }

  @override
  void dispose() {
    _speaker.stop();
    super.dispose();
  }
}
