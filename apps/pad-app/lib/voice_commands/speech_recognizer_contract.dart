class SpeechTranscript {
  const SpeechTranscript({
    required this.transcript,
    required this.locale,
  });

  final String transcript;
  final String locale;
}

typedef SpeechTranscriptListener = void Function(String transcript);
typedef SpeechSegmentListener = void Function(String segment);

abstract interface class SpeechRecognizer {
  bool get supportsRecognition;
  bool get isListening;

  Future<void> startListening({
    required String locale,
    SpeechTranscriptListener? onTranscriptChanged,
    SpeechSegmentListener? onSegmentCommitted,
  });

  Future<SpeechTranscript> finishListening();

  Future<SpeechTranscript> listenOnce({
    required String locale,
  });

  Future<void> stop();
}
