import 'package:pad_app/word_playback/models.dart';

abstract interface class WordSpeaker {
  bool get supportsPlayback;

  Future<void> speak(
    String text, {
    required WordPlaybackLanguage language,
    double? speechRate,
    double? pitch,
  });

  Future<void> stop();
}
