import 'dart:async';

import 'package:pad_app/voice_commands/speech_recognizer_contract.dart';
import 'package:speech_to_text/speech_recognition_error.dart';
import 'package:speech_to_text/speech_recognition_result.dart';
import 'package:speech_to_text/speech_to_text.dart';

SpeechRecognizer createSpeechRecognizer() {
  return SpeechToTextRecognizer();
}

class SpeechToTextRecognizer implements SpeechRecognizer {
  final SpeechToText _speech = SpeechToText();

  Future<bool>? _initializeFuture;
  Completer<SpeechTranscript>? _listenCompleter;
  String? _activeLocale;
  bool _supportsRecognition = true;
  bool _isListening = false;

  @override
  bool get supportsRecognition => _supportsRecognition;

  @override
  bool get isListening => _isListening;

  @override
  Future<SpeechTranscript> listenOnce({
    required String locale,
  }) async {
    final available = await _ensureInitialized();
    if (!available) {
      throw UnsupportedError('当前设备或浏览器不支持语音识别。');
    }
    if (_isListening) {
      throw StateError('语音识别正在进行中。');
    }

    _activeLocale = locale;
    _isListening = true;
    final completer = Completer<SpeechTranscript>();
    _listenCompleter = completer;

    final started = await _speech.listen(
      onResult: _onResult,
      listenFor: const Duration(seconds: 8),
      pauseFor: const Duration(seconds: 2),
      localeId: locale,
      listenOptions: SpeechListenOptions(
        partialResults: false,
        cancelOnError: true,
      ),
    );
    if (!started) {
      _resetListeningState();
      throw StateError('没有成功启动语音识别，请再试一次。');
    }

    return completer.future.whenComplete(() async {
      await _speech.stop();
      _resetListeningState();
    });
  }

  @override
  Future<void> stop() async {
    if (!_isListening) {
      return;
    }
    await _speech.stop();
    final completer = _listenCompleter;
    _resetListeningState();
    if (completer != null && !completer.isCompleted) {
      completer.completeError(StateError('语音识别已取消。'));
    }
  }

  Future<bool> _ensureInitialized() {
    return _initializeFuture ??= _speech
        .initialize(
      onError: _onError,
      onStatus: _onStatus,
      debugLogging: false,
    )
        .then((value) {
      _supportsRecognition = value;
      return value;
    });
  }

  void _onResult(SpeechRecognitionResult result) {
    final completer = _listenCompleter;
    if (completer == null || completer.isCompleted || !result.finalResult) {
      return;
    }

    final transcript = result.recognizedWords.trim();
    if (transcript.isEmpty) {
      completer.completeError(StateError('没有听到有效语音，请再试一次。'));
      return;
    }

    completer.complete(
      SpeechTranscript(
        transcript: transcript,
        locale: _activeLocale ?? '',
      ),
    );
  }

  void _onError(SpeechRecognitionError error) {
    final completer = _listenCompleter;
    if (completer == null || completer.isCompleted) {
      return;
    }
    completer.completeError(StateError('语音识别失败：${error.errorMsg}'));
  }

  void _onStatus(String status) {
    final completer = _listenCompleter;
    if (completer == null || completer.isCompleted) {
      return;
    }

    if (status == 'done' || status == 'notListening') {
      completer.completeError(StateError('没有听到有效语音，请再试一次。'));
    }
  }

  void _resetListeningState() {
    _isListening = false;
    _activeLocale = null;
    _listenCompleter = null;
  }
}
