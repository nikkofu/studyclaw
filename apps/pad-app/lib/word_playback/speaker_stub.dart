import 'package:pad_app/word_playback/models.dart';
import 'package:pad_app/word_playback/speaker_contract.dart';

WordSpeaker createWordSpeaker() {
  return const UnsupportedWordSpeaker();
}

class UnsupportedWordSpeaker implements WordSpeaker {
  const UnsupportedWordSpeaker();

  @override
  bool get supportsPlayback => false;

  @override
  Future<void> speak(
    String text, {
    required WordPlaybackLanguage language,
    double? speechRate,
    double? pitch,
  }) async {}

  @override
  Future<void> stop() async {}
}
