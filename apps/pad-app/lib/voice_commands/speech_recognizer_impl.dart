import 'dart:async';

import 'package:pad_app/voice_commands/speech_recognizer_contract.dart';
import 'package:speech_to_text/speech_recognition_error.dart';
import 'package:speech_to_text/speech_recognition_result.dart';
import 'package:speech_to_text/speech_to_text.dart';

SpeechRecognizer createSpeechRecognizer() {
  return SpeechToTextRecognizer();
}

class SpeechToTextRecognizer implements SpeechRecognizer {
  SpeechToTextRecognizer({
    SpeechToText? speech,
  }) : _speech = speech ?? SpeechToText();

  final SpeechToText _speech;

  Future<bool>? _initializeFuture;
  Completer<SpeechTranscript>? _listenCompleter;
  String? _activeLocale;
  String _latestTranscript = '';
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
    _latestTranscript = '';
    _isListening = true;
    final completer = Completer<SpeechTranscript>();
    _listenCompleter = completer;

    await _speech.listen(
      onResult: _onResult,
      listenFor: const Duration(seconds: 8),
      pauseFor: const Duration(seconds: 2),
      localeId: locale,
      listenOptions: SpeechListenOptions(
        partialResults: false,
        cancelOnError: true,
      ),
    );

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
    if (completer == null || completer.isCompleted) {
      return;
    }

    final transcript = result.recognizedWords.trim();
    if (transcript.isNotEmpty) {
      _latestTranscript = transcript;
    }

    if (!result.finalResult) {
      return;
    }

    _completeTranscript(completer, transcript);
  }

  void _completeTranscript(
    Completer<SpeechTranscript> completer,
    String transcript,
  ) {
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
      _completeTranscript(completer, _latestTranscript.trim());
    }
  }

  void _resetListeningState() {
    _isListening = false;
    _activeLocale = null;
    _latestTranscript = '';
    _listenCompleter = null;
  }
}
