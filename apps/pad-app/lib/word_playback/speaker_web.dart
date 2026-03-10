import 'package:pad_app/word_playback/models.dart';
import 'package:pad_app/word_playback/speaker_contract.dart';
import 'package:web/web.dart' as web;

WordSpeaker createWordSpeaker() {
  return BrowserWordSpeaker();
}

class BrowserWordSpeaker implements WordSpeaker {
  @override
  bool get supportsPlayback => true;

  @override
  Future<void> speak(
    String text, {
    required WordPlaybackLanguage language,
  }) async {
    final utterance = web.SpeechSynthesisUtterance(text)
      ..lang = language.localeCode
      ..rate = 0.9;

    web.window.speechSynthesis.cancel();
    web.window.speechSynthesis.speak(utterance);
  }

  @override
  Future<void> stop() async {
    web.window.speechSynthesis.cancel();
  }
}
