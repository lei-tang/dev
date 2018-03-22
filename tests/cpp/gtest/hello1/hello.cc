#include <gtest/gtest.h>
#include <iostream>
#include <string>

using namespace std;

string getGreeting() { return string("Hello"); }

string getGreetingToSubject(string sub) { return string("Hello: " + sub); }

TEST(GreetingTest, GetGreetingMatchHello) {
  EXPECT_STREQ("Hello", getGreeting().c_str());
}

TEST(GreetingTest, GetGreetingMatchSubject) {
  EXPECT_STREQ("Hello: Tom", getGreetingToSubject("Tom").c_str());
}

int main(int argc, char **argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
