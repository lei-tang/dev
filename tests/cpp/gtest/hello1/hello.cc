#include <gtest/gtest.h>
#include <iostream>
#include <string>

using namespace std;

string getGreeting() { return string("Hi"); }

string getGreetingToSubject(string sub) { return string("Hi, " + sub); }

TEST(GreetingTest, GetGreetingMatchHello) {
  EXPECT_STREQ("Hello", getGreeting().c_str());
}

TEST(GreetingTest, GetGreetingMatchSubject) {
  EXPECT_STREQ("Hello: Jack", getGreetingToSubject("Jack").c_str());
}

int main(int argc, char **argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
