import 'package:pad_app/word_playback/speaker_contract.dart';
import 'package:pad_app/word_playback/speaker_stub.dart'
    if (dart.library.html) 'package:pad_app/word_playback/speaker_web.dart'
    if (dart.library.io) 'package:pad_app/word_playback/speaker_web.dart'
    as implementation;

WordSpeaker createWordSpeaker() {
  return implementation.createWordSpeaker();
}
