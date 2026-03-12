class SpeechTranscript {
  const SpeechTranscript({
    required this.transcript,
    required this.locale,
  });

  final String transcript;
  final String locale;
}

abstract interface class SpeechRecognizer {
  bool get supportsRecognition;
  bool get isListening;

  Future<SpeechTranscript> listenOnce({
    required String locale,
  });

  Future<void> stop();
}
