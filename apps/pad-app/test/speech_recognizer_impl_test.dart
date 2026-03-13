import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:pad_app/voice_commands/speech_recognizer_impl.dart';
import 'package:speech_to_text/speech_recognition_result.dart';
import 'package:speech_to_text/speech_to_text.dart';

class _FakeSpeechToText extends SpeechToText {
  _FakeSpeechToText({
    required this.onListen,
  }) : super.withMethodChannel();

  final Future<void> Function(
    SpeechResultListener? onResult,
    SpeechStatusListener? onStatus,
  ) onListen;

  SpeechStatusListener? _statusListener;
  bool stopCalled = false;

  @override
  Future<bool> initialize({
    SpeechErrorListener? onError,
    SpeechStatusListener? onStatus,
    debugLogging = false,
    Duration finalTimeout = SpeechToText.defaultFinalTimeout,
    List<SpeechConfigOption>? options,
  }) async {
    _statusListener = onStatus;
    return true;
  }

  @override
  Future listen({
    SpeechResultListener? onResult,
    Duration? listenFor,
    Duration? pauseFor,
    String? localeId,
    SpeechSoundLevelChange? onSoundLevelChange,
    cancelOnError = false,
    partialResults = true,
    onDevice = false,
    ListenMode listenMode = ListenMode.confirmation,
    sampleRate = 0,
    SpeechListenOptions? listenOptions,
  }) async {
    scheduleMicrotask(() {
      unawaited(onListen(onResult, _statusListener));
    });
  }

  @override
  Future<void> stop() async {
    stopCalled = true;
  }
}

void main() {
  test('completes from a web-style interim result followed by done status', () async {
    final speech = _FakeSpeechToText(
      onListen: (onResult, onStatus) async {
        onResult?.call(
          SpeechRecognitionResult(
            const [
              SpeechRecognitionWords('数学订正好了', 0.92),
            ],
            false,
          ),
        );
        onStatus?.call('done');
      },
    );

    final recognizer = SpeechToTextRecognizer(speech: speech);
    final transcript = await recognizer.listenOnce(locale: 'zh-CN');

    expect(transcript.transcript, '数学订正好了');
    expect(transcript.locale, 'zh-CN');
    expect(speech.stopCalled, isTrue);
  });

  test('throws a helpful error when done arrives without any transcript', () async {
    final speech = _FakeSpeechToText(
      onListen: (onResult, onStatus) async {
        onStatus?.call('done');
      },
    );

    final recognizer = SpeechToTextRecognizer(speech: speech);

    await expectLater(
      recognizer.listenOnce(locale: 'zh-CN'),
      throwsA(
        isA<StateError>().having(
          (error) => error.message,
          'message',
          '没有听到有效语音，请再试一次。',
        ),
      ),
    );
    expect(speech.stopCalled, isTrue);
  });
}
