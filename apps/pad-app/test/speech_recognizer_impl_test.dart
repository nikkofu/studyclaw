import 'dart:async';
import 'dart:collection';

import 'package:flutter_test/flutter_test.dart';
import 'package:pad_app/voice_commands/speech_recognizer_impl.dart';
import 'package:pad_app/voice_commands/speech_recognizer_contract.dart';
import 'package:speech_to_text/speech_recognition_error.dart';
import 'package:speech_to_text/speech_recognition_result.dart';
import 'package:speech_to_text/speech_to_text.dart';

typedef _ListenHandler = Future<void> Function(
  SpeechResultListener? onResult,
  SpeechStatusListener? onStatus,
  SpeechErrorListener? onError,
);

class _FakeSpeechToText extends SpeechToText {
  _FakeSpeechToText({
    required List<_ListenHandler> listenHandlers,
    this.onStop,
  })  : _listenHandlers = Queue.of(listenHandlers),
        super.withMethodChannel();

  final Queue<_ListenHandler> _listenHandlers;
  final Future<void> Function(SpeechStatusListener? onStatus)? onStop;

  SpeechStatusListener? _statusListener;
  SpeechErrorListener? _errorListener;
  bool stopCalled = false;
  int listenCallCount = 0;
  bool? lastCancelOnError;

  @override
  Future<bool> initialize({
    SpeechErrorListener? onError,
    SpeechStatusListener? onStatus,
    debugLogging = false,
    Duration finalTimeout = SpeechToText.defaultFinalTimeout,
    List<SpeechConfigOption>? options,
  }) async {
    _statusListener = onStatus;
    _errorListener = onError;
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
    listenCallCount += 1;
    lastCancelOnError = listenOptions?.cancelOnError;
    final handler =
        _listenHandlers.isNotEmpty ? _listenHandlers.removeFirst() : null;
    scheduleMicrotask(() {
      if (handler != null) {
        unawaited(handler(onResult, _statusListener, _errorListener));
      }
    });
  }

  @override
  Future<void> stop() async {
    stopCalled = true;
    if (onStop != null) {
      await onStop!(_statusListener);
    }
  }
}

void main() {
  test('completes from a web-style interim result followed by done status',
      () async {
    final speech = _FakeSpeechToText(
      listenHandlers: [
        (onResult, onStatus, onError) async {
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
      ],
    );

    final recognizer = SpeechToTextRecognizer(speech: speech);
    final transcript = await recognizer.listenOnce(locale: 'zh-CN');

    expect(transcript.transcript, '数学订正好了');
    expect(transcript.locale, 'zh-CN');
    expect(speech.stopCalled, isTrue);
  });

  test('throws a helpful error when done arrives without any transcript',
      () async {
    final speech = _FakeSpeechToText(
      listenHandlers: [
        (onResult, onStatus, onError) async {
          onStatus?.call('done');
        },
      ],
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

  test('keeps listening across pauses until finishListening is called',
      () async {
    final transcriptUpdates = <String>[];
    final segmentCommits = <String>[];
    final speech = _FakeSpeechToText(
      listenHandlers: [
        (onResult, onStatus, onError) async {
          onResult?.call(
            SpeechRecognitionResult(
              const [
                SpeechRecognitionWords('第一段', 0.91),
              ],
              false,
            ),
          );
          onStatus?.call('done');
        },
        (onResult, onStatus, onError) async {
          onResult?.call(
            SpeechRecognitionResult(
              const [
                SpeechRecognitionWords('第二段', 0.93),
              ],
              false,
            ),
          );
        },
      ],
      onStop: (onStatus) async {
        onStatus?.call('done');
      },
    );

    final recognizer = SpeechToTextRecognizer(speech: speech);
    await recognizer.startListening(
      locale: 'zh-CN',
      onTranscriptChanged: transcriptUpdates.add,
      onSegmentCommitted: segmentCommits.add,
    );

    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);

    expect(recognizer.isListening, isTrue);
    expect(speech.listenCallCount, 2);
    expect(transcriptUpdates, contains('第一段'));
    expect(segmentCommits, contains('第一段'));

    final transcript = await recognizer.finishListening();

    expect(transcript, isA<SpeechTranscript>());
    expect(transcript.transcript, '第一段 第二段');
    expect(transcript.locale, 'zh-CN');
    expect(segmentCommits, <String>['第一段', '第二段']);
    expect(speech.stopCalled, isTrue);
    expect(recognizer.isListening, isFalse);
  });

  test('keeps restarting through silent cycles before speech arrives',
      () async {
    final transcriptUpdates = <String>[];
    final segmentCommits = <String>[];
    final speech = _FakeSpeechToText(
      listenHandlers: [
        (onResult, onStatus, onError) async {
          onStatus?.call('done');
        },
        (onResult, onStatus, onError) async {
          onError?.call(SpeechRecognitionError('error_no_match', false));
        },
        (onResult, onStatus, onError) async {
          onStatus?.call('done');
        },
        (onResult, onStatus, onError) async {
          onResult?.call(
            SpeechRecognitionResult(
              const [
                SpeechRecognitionWords('现在开始背诵', 0.95),
              ],
              false,
            ),
          );
          onStatus?.call('done');
        },
      ],
      onStop: (onStatus) async {
        onStatus?.call('done');
      },
    );

    final recognizer = SpeechToTextRecognizer(speech: speech);
    await recognizer.startListening(
      locale: 'zh-CN',
      onTranscriptChanged: transcriptUpdates.add,
      onSegmentCommitted: segmentCommits.add,
    );

    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);

    expect(recognizer.isListening, isTrue);
    expect(speech.listenCallCount, greaterThanOrEqualTo(3));
    expect(speech.lastCancelOnError, isFalse);
    expect(transcriptUpdates, contains('现在开始背诵'));
    expect(segmentCommits, <String>['现在开始背诵']);

    final transcript = await recognizer.finishListening();
    expect(transcript.transcript, '现在开始背诵');
    expect(transcript.locale, 'zh-CN');
    expect(speech.stopCalled, isTrue);
  });

  test('recovers from transient no-match errors in continuous listening',
      () async {
    final transcriptUpdates = <String>[];
    final segmentCommits = <String>[];
    final speech = _FakeSpeechToText(
      listenHandlers: [
        (onResult, onStatus, onError) async {
          onError?.call(SpeechRecognitionError('error_no_match', false));
        },
        (onResult, onStatus, onError) async {
          onResult?.call(
            SpeechRecognitionResult(
              const [
                SpeechRecognitionWords('桃花一簇开无主', 0.95),
              ],
              false,
            ),
          );
          onStatus?.call('done');
        },
      ],
      onStop: (onStatus) async {
        onStatus?.call('done');
      },
    );

    final recognizer = SpeechToTextRecognizer(speech: speech);
    await recognizer.startListening(
      locale: 'zh-CN',
      onTranscriptChanged: transcriptUpdates.add,
      onSegmentCommitted: segmentCommits.add,
    );

    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);

    expect(recognizer.isListening, isTrue);
    expect(speech.listenCallCount, greaterThanOrEqualTo(2));
    expect(transcriptUpdates, contains('桃花一簇开无主'));
    expect(segmentCommits, <String>['桃花一簇开无主']);

    final transcript = await recognizer.finishListening();
    expect(transcript.transcript, '桃花一簇开无主');
    expect(speech.stopCalled, isTrue);
  });

  test('still fails fast on permanent recognition errors', () async {
    final speech = _FakeSpeechToText(
      listenHandlers: [
        (onResult, onStatus, onError) async {
          onError?.call(SpeechRecognitionError('error_not_allowed', true));
        },
      ],
    );

    final recognizer = SpeechToTextRecognizer(speech: speech);

    await expectLater(
      recognizer.listenOnce(locale: 'zh-CN'),
      throwsA(
        isA<StateError>().having(
          (error) => error.message,
          'message',
          '语音识别失败：error_not_allowed',
        ),
      ),
    );
    expect(speech.stopCalled, isTrue);
  });
}
