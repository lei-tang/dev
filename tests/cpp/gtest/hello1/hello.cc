#include <gtest/gtest.h>
#include <iostream>
#include <string>

using namespace std;

string getGreeting() { return string("Hello"); }

TEST(GreetingTest, GetGreetingMatchHello) {
  EXPECT_STREQ("Hello", getGreeting().c_str());
}

int main(int argc, char **argv) {
  ::testing::InitGoogleTest(&argc, argv);
  return RUN_ALL_TESTS();
}
