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
  SpeechTranscriptListener? _transcriptListener;
  SpeechSegmentListener? _segmentListener;
  String _committedTranscript = '';
  String _latestTranscript = '';
  bool _supportsRecognition = true;
  bool _isListening = false;
  bool _isContinuousSession = false;
  bool _isFinishing = false;
  bool _restartScheduled = false;

  static const List<String> _recoverableContinuousErrorFragments = <String>[
    'error_no_match',
    'error_speech_timeout',
    'no match',
    'no speech',
    'speech timeout',
    'timeout',
    'aborted',
  ];

  @override
  bool get supportsRecognition => _supportsRecognition;

  @override
  bool get isListening => _isListening;

  @override
  Future<void> startListening({
    required String locale,
    SpeechTranscriptListener? onTranscriptChanged,
    SpeechSegmentListener? onSegmentCommitted,
  }) async {
    await _beginListening(
      locale: locale,
      continuous: true,
      onTranscriptChanged: onTranscriptChanged,
      onSegmentCommitted: onSegmentCommitted,
    );
  }

  @override
  Future<SpeechTranscript> finishListening() async {
    final completer = _listenCompleter;
    if (!_isListening || completer == null || completer.isCompleted) {
      throw StateError('语音识别还没有开始。');
    }

    _isFinishing = true;
    await _speech.stop();

    return completer.future.whenComplete(() async {
      await _speech.stop();
      _resetListeningState();
    });
  }

  @override
  Future<SpeechTranscript> listenOnce({
    required String locale,
  }) async {
    await _beginListening(locale: locale, continuous: false);
    final completer = _listenCompleter!;

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

  Future<void> _beginListening({
    required String locale,
    required bool continuous,
    SpeechTranscriptListener? onTranscriptChanged,
    SpeechSegmentListener? onSegmentCommitted,
  }) async {
    final available = await _ensureInitialized();
    if (!available) {
      throw UnsupportedError('当前设备或浏览器不支持语音识别。');
    }
    if (_isListening) {
      throw StateError('语音识别正在进行中。');
    }

    _activeLocale = locale;
    _transcriptListener = onTranscriptChanged;
    _segmentListener = onSegmentCommitted;
    _committedTranscript = '';
    _latestTranscript = '';
    _isListening = true;
    _isContinuousSession = continuous;
    _isFinishing = false;
    _restartScheduled = false;
    _listenCompleter = Completer<SpeechTranscript>();

    await _startListenCycle();
  }

  Future<void> _startListenCycle() async {
    await _speech.listen(
      onResult: _onResult,
      listenFor: _isContinuousSession
          ? const Duration(minutes: 5)
          : const Duration(seconds: 8),
      pauseFor: _isContinuousSession
          ? const Duration(seconds: 45)
          : const Duration(seconds: 2),
      localeId: _activeLocale,
      listenOptions: SpeechListenOptions(
        partialResults: true,
        cancelOnError: !_isContinuousSession,
        listenMode: _isContinuousSession
            ? ListenMode.dictation
            : ListenMode.confirmation,
      ),
    );
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
      _notifyTranscriptChanged();
    }

    if (!result.finalResult) {
      return;
    }

    _commitLatestTranscript();
    if (!_isContinuousSession) {
      _completeTranscript(completer, _committedTranscript.trim());
    }
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
    if (_shouldRecoverFromError(error)) {
      _commitLatestTranscript();
      _scheduleRestart();
      return;
    }
    completer.completeError(StateError('语音识别失败：${error.errorMsg}'));
  }

  bool _shouldRecoverFromError(SpeechRecognitionError error) {
    if (!_isContinuousSession || _isFinishing) {
      return false;
    }
    if (!error.permanent) {
      return true;
    }
    final normalizedError = error.errorMsg.trim().toLowerCase();
    return _recoverableContinuousErrorFragments.any(
      normalizedError.contains,
    );
  }

  void _onStatus(String status) {
    final completer = _listenCompleter;
    if (completer == null || completer.isCompleted) {
      return;
    }

    if (status == 'done' || status == 'notListening') {
      _commitLatestTranscript();

      if (_isContinuousSession) {
        if (_isFinishing) {
          _completeTranscript(completer, _committedTranscript.trim());
          return;
        }

        _scheduleRestart();
        return;
      }

      _completeTranscript(completer, _committedTranscript.trim());
    }
  }

  void _scheduleRestart() {
    if (_restartScheduled || !_isListening || _isFinishing) {
      return;
    }

    _restartScheduled = true;
    scheduleMicrotask(() async {
      _restartScheduled = false;
      final completer = _listenCompleter;
      if (!_isListening ||
          _isFinishing ||
          completer == null ||
          completer.isCompleted) {
        return;
      }
      if (_speech.isListening) {
        return;
      }

      try {
        await _startListenCycle();
      } catch (error) {
        if (completer.isCompleted) {
          return;
        }
        completer.completeError(StateError('语音识别失败：$error'));
      }
    });
  }

  void _commitLatestTranscript() {
    final transcript = _latestTranscript.trim();
    if (transcript.isEmpty) {
      return;
    }

    String? committedSegment;
    if (_committedTranscript.isEmpty) {
      _committedTranscript = transcript;
      committedSegment = transcript;
    } else if (_committedTranscript == transcript ||
        _committedTranscript.endsWith(transcript)) {
      // The plugin may resend the same content when a listen cycle ends.
    } else if (transcript.startsWith(_committedTranscript)) {
      committedSegment =
          transcript.substring(_committedTranscript.length).trim();
      _committedTranscript = transcript;
    } else {
      _committedTranscript = '$_committedTranscript $transcript'.trim();
      committedSegment = transcript;
    }

    _latestTranscript = '';
    if (committedSegment != null && committedSegment.isNotEmpty) {
      _segmentListener?.call(committedSegment);
    }
    _notifyTranscriptChanged();
  }

  void _notifyTranscriptChanged() {
    final listener = _transcriptListener;
    if (listener == null) {
      return;
    }

    final preview = _buildPreviewTranscript();
    if (preview.isEmpty) {
      return;
    }
    listener(preview);
  }

  String _buildPreviewTranscript() {
    final latest = _latestTranscript.trim();
    if (_committedTranscript.isEmpty) {
      return latest;
    }
    if (latest.isEmpty) {
      return _committedTranscript;
    }
    if (_committedTranscript.endsWith(latest)) {
      return _committedTranscript;
    }
    if (latest.startsWith(_committedTranscript)) {
      return latest;
    }
    return '$_committedTranscript $latest'.trim();
  }

  void _resetListeningState() {
    _isListening = false;
    _isContinuousSession = false;
    _isFinishing = false;
    _restartScheduled = false;
    _activeLocale = null;
    _transcriptListener = null;
    _segmentListener = null;
    _committedTranscript = '';
    _latestTranscript = '';
    _listenCompleter = null;
  }
}
